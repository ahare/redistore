# redistore

A session store backend for [gorilla/sessions](http://www.gorillatoolkit.org/pkg/sessions) - [src](https://github.com/gorilla/sessions). This is a modified version of [boj/redistore](https://github.com/boj/redistore) in that you pass an existing redis connection rather than allowing redistore to create connections for you.

## Requirements

Depends on the [Redigo](https://github.com/garyburd/redigo) Redis library.

## Installation

    go get github.com/ahare/redistore

## Documentation

Available on [godoc.org](http://www.godoc.org/github.com/ahare/redistore).

See http://www.gorillatoolkit.org/pkg/sessions for full documentation on underlying interface.

### Example

    // Fetch new store.
    conn := redis.Dial("tcp", ":6379")
    store := NewRediStore(conn, []byte("secret-key"))
    defer store.Close()

    // Get a session.
	session, err = store.Get(req, "session-key")
	if err != nil {
        log.Error(err.Error())
    }

    // Add a value.
    session.Values["foo"] = "bar"

    // Save.
    if err = sessions.Save(req, rsp); err != nil {
        t.Fatalf("Error saving session: %v", err)
    }

    // Delete session.
    session.Options.MaxAge = -1
    if err = sessions.Save(req, rsp); err != nil {
        t.Fatalf("Error saving session: %v", err)
    }

## Notes

#### July 18th, 2013

* __Delete()__ should be considered deprecated since it is not exposed via the gorilla/sessions interface.  Set session.Options.MaxAge = -1 and call Save instead.
