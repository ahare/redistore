// Copyright 2012 Brian "bojo" Jones. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package redistore

import (
	"encoding/base32"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"net/http"
	"strings"
)

// Amount of time for cookies/redis keys to expire.
var sessionExpire int = 86400 * 30

// RediStore stores sessions in a redis backend.
type RediStore struct {
	Conn    redis.Conn
	Codecs  []securecookie.Codec
	Options *sessions.Options // default configuration
}

// NewRediStore returns a new RediStore.
func NewRediStore(conn redis.Conn, keyPairs ...[]byte) *RediStore {
	return &RediStore{
		Conn:   conn,
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: sessionExpire,
		},
	}
}

// Close cleans up the redis connections.
func (s *RediStore) Close() {
	s.Conn.Close()
}

// Get returns a session for the given name after adding it to the registry.
//
// See gorilla/sessions FilesystemStore.Get().
func (s *RediStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
//
// See gorilla/sessions FilesystemStore.New().
func (s *RediStore) New(r *http.Request, name string) (*sessions.Session, error) {
	var err error
	session := sessions.NewSession(s, name)
	session.Options = &(*s.Options)
	session.IsNew = true
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			err = s.load(session)
			if err == nil {
				session.IsNew = false
			}
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *RediStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Marked for deletion.
	if session.Options.MaxAge < 0 {
		if err := s.delete(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
	} else {
		// Build an alphanumeric key for the redis store.
		if session.ID == "" {
			session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
		}
		if err := s.save(session); err != nil {
			return err
		}
		encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
		if err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	}
	return nil
}

// Delete removes the session from redis, and sets the cookie to expire.
//
// WARNING: This method should be considered deprecated since it is not exposed via the gorilla/sessions interface.
// Set session.Options.MaxAge = -1 and call Save instead. - July 18th, 2013
func (s *RediStore) Delete(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	if _, err := s.Conn.Do("DEL", "session_"+session.ID); err != nil {
		return err
	}
	// Set cookie to expire.
	options := *session.Options
	options.MaxAge = -1
	http.SetCookie(w, sessions.NewCookie(session.Name(), "", &options))
	// Clear session values.
	for k := range session.Values {
		delete(session.Values, k)
	}
	return nil
}

// save stores the session in redis.
func (s *RediStore) save(session *sessions.Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, s.Codecs...)
	if err != nil {
		return err
	}
	s.Conn.Send("SET", "session_"+session.ID, encoded)
	s.Conn.Send("EXPIRE", "session_"+session.ID, sessionExpire)
	s.Conn.Flush()
	if _, err := s.Conn.Receive(); err != nil { // SET
		return err
	}
	if _, err := s.Conn.Receive(); err != nil { // EXPIRE
		return err
	}
	return nil
}

// load reads the session from redis.
func (s *RediStore) load(session *sessions.Session) error {
	if err := s.Conn.Err(); err != nil {
		return err
	}
	data, err := s.Conn.Do("GET", "session_"+session.ID)
	if err != nil {
		return err
	}
	if data == nil {
		return nil // no data was associated with this key
	}
	str, err := redis.String(data, err)
	if err != nil {
		return err
	}
	if err = securecookie.DecodeMulti(session.Name(), str, &session.Values, s.Codecs...); err != nil {
		return err
	}
	return nil
}

// delete removes keys from redis if MaxAge<0
func (s *RediStore) delete(session *sessions.Session) error {
	if _, err := s.Conn.Do("DEL", "session_"+session.ID); err != nil {
		return err
	}
	return nil
}
