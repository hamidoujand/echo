package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hamidoujand/echo/errs"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type users interface {
	Add(usr User) error
	Remove(userID uuid.UUID)
	Retrieve(userID uuid.UUID) (User, error)
	Connections() map[uuid.UUID]Connection
	UpdateLastPong(usrID uuid.UUID) (User, error)
	UpdateLastPing(usrID uuid.UUID) error
}

type Chat struct {
	capID    string
	log      *slog.Logger
	users    users
	js       jetstream.JetStream
	consumer jetstream.Consumer
	stream   jetstream.Stream
	subject  string
}

func New(log *slog.Logger, users users, conn *nats.Conn, subject string, capID string) (*Chat, error) {
	//create jetstream
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("create jetStream: %w", err)
	}

	ctx := context.Background()

	//create a stream
	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     subject,
		Subjects: []string{subject},
		MaxAge:   20 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("creating stream: %w", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       capID,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})

	if err != nil {
		return nil, fmt.Errorf("creating a jetstream consumer: %w", err)
	}

	c := Chat{
		capID:    capID,
		log:      log,
		users:    users,
		js:       js,
		consumer: consumer,
		stream:   stream,
		subject:  subject,
	}

	const maxWait = time.Second * 10
	c.ping(maxWait)
	c.listenBUS()

	return &c, nil
}

func (c *Chat) Handshake(ctx context.Context, w http.ResponseWriter, r *http.Request) (User, error) {
	var ws websocket.Upgrader
	conn, err := ws.Upgrade(w, r, nil)
	if err != nil {
		return User{}, errs.New(http.StatusBadRequest, fmt.Errorf("upgrade failed: %w", err))
	}

	//write to the connection
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Hello")); err != nil {
		return User{}, fmt.Errorf("writing message to conn: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	usr := User{
		Conn:     conn,
		LastPing: time.Now(),
		LastPong: time.Now(),
	}

	msg, err := c.readMessage(ctx, usr)
	if err != nil {
		return User{}, fmt.Errorf("reading message: %w", err)
	}

	if err := json.Unmarshal(msg, &usr); err != nil {
		return User{}, fmt.Errorf("unmarshal msg: %w", err)
	}

	//add user
	if err := c.users.Add(usr); err != nil {
		defer func() { _ = conn.Close() }()

		//user already exists,close the new connection
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Already connected")); err != nil {
			return User{}, fmt.Errorf("writing message to conn: %w", err)
		}

		return User{}, fmt.Errorf("adding user: %w", err)
	}

	usr.Conn.SetPongHandler(c.pong(usr.ID))
	//send an ack
	ack := fmt.Sprintf("Welcome, %s", usr.Name)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(ack)); err != nil {
		return User{}, fmt.Errorf("writing message: %w", err)
	}

	c.log.Info("handshake completed", "id", usr.ID)

	return usr, nil
}

func (c *Chat) Listen(ctx context.Context, usr User) {
	for {
		msg, err := c.readMessage(ctx, usr)
		if err != nil {
			switch v := err.(type) {
			case *websocket.CloseError:
				c.log.Error("client disconnected", "status", "reading message", "err", err)
				return
			case *net.OpError:
				if !v.Temporary() {
					c.log.Error("client disconnected", "status", "reading message", "err", err)
					return
				}
				//if its temporary
				continue
			default:
				//check for context cancelled
				if errors.Is(err, context.Canceled) {
					c.log.Error("client context is cancelled", "status", "reading message", "err", err)
					return
				}

				c.log.Error("error while reading message", "err", err)
				continue
			}
		}

		//create the inMessage
		var in inMessage
		if err := json.Unmarshal(msg, &in); err != nil {
			c.log.Error("unmarshaling inMessage failed", "err", err)
			continue
		}

		c.log.Info("received message", "from", usr.ID, "to", in.ToID, "msg type", websocket.TextMessage)

		to, err := c.users.Retrieve(in.ToID)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				//send to the BUS
				m := busMessage{
					CapID:    c.capID,
					FromID:   usr.ID,
					FromName: usr.Name,
					ToID:     in.ToID,
					Text:     in.Text,
				}

				c.sendMessageToBUS(ctx, m)
			} else {
				c.log.Error("failed to retrieve the message's recipient", "err", err)
			}

			continue
		}

		if err := c.sendMessage(usr, to, in.Text); err != nil {
			c.log.Error("sending message failed", "err", err)
		}

		c.log.Info("sent message", "from", usr.ID, "to", in.ToID)
	}
}

func (c *Chat) pong(usrID uuid.UUID) func(appData string) error {
	h := func(appData string) error {
		usr, err := c.users.UpdateLastPong(usrID)
		if err != nil {
			c.log.Error("updating user's lastPong failed", "err", err, "id", usrID)
			return nil
		}

		diff := usr.LastPong.Sub(usr.LastPing)
		c.log.Debug("pong handler", "id", usr.ID, "took", diff)

		return nil
	}

	return h
}

func (c *Chat) ping(maxWait time.Duration) {
	ticker := time.NewTicker(maxWait)
	defer ticker.Stop()

	go func() {
		for {
			//block for the tick, then ping all connections.
			<-ticker.C
			connections := c.users.Connections()

			for id, conn := range connections {
				diff := conn.LastPong.Sub(conn.LastPing)
				if diff > maxWait {
					//remove it
					c.log.Error("duration between ping and pong is greater the maxWaiting time",
						"ping", conn.LastPing.String(),
						"pong", conn.LastPong.String(),
						"maxWait", maxWait,
						"diff", diff.String(),
					)
					c.users.Remove(id)
					continue
				}

				if err := conn.Conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.log.Error("sending ping failed", "id", id, "err", err)
				}

				c.log.Debug("ping handler,sent ping", "id", id)

				if err := c.users.UpdateLastPing(id); err != nil {
					c.log.Error("updating last ping failed", "id", id, "err", err)
				}
			}
		}
	}()
}

func (c *Chat) readMessage(ctx context.Context, usr User) ([]byte, error) {
	type response struct {
		msg []byte
		err error
	}

	ch := make(chan response, 1)
	go func() {
		_, msg, err := usr.Conn.ReadMessage()
		if err != nil {
			ch <- response{msg: nil, err: err}
		}

		ch <- response{msg: msg, err: nil}
	}()

	select {
	case <-ctx.Done():
		c.users.Remove(usr.ID)
		usr.Conn.Close()
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.err != nil {
			c.users.Remove(usr.ID)
			usr.Conn.Close()
			return nil, resp.err
		}

		return resp.msg, nil
	}
}

func (c *Chat) sendMessage(from User, to User, msg string) error {
	m := outMessage{
		From: User{ID: from.ID, Name: from.Name},
		Text: msg,
	}

	if err := to.Conn.WriteJSON(m); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}

	return nil
}

func (c *Chat) sendMessageToBUS(ctx context.Context, msg busMessage) error {
	bs, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshalling msg: %w", err)
	}

	_, err = c.js.Publish(ctx, c.subject, bs)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func (c *Chat) listenBUS() {
	go func() {
		for {
			msg, err := c.readMessageFromBUS(context.Background())
			if err != nil {
				//critical errors
				if errors.Is(err, context.Canceled) {
					c.log.Error("client canceled the context", "err", err)
					return
				} else if errors.Is(err, nats.ErrConnectionClosed) {
					c.log.Error("nats connection is closed", "err", err)
					return
				}
				//non-critical
				c.log.Error("listenBUS: error while reading message", "err", err)
				continue
			}

			//create the inMessage
			var bm busMessage
			if err := json.Unmarshal(msg.Data(), &bm); err != nil {
				c.log.Error("unmarshaling BUS message failed", "err", err)
				continue
			}
			//skip our own messages
			if bm.CapID == c.capID {
				continue
			}
			c.log.Info("received message from BUS", "from", bm.FromID, "to", bm.ToID, "msg type", websocket.TextMessage)

			to, err := c.users.Retrieve(bm.ToID)
			if err != nil {
				//not found in this cap
				c.log.Error("listenBUS: recipient is not found in this CAP", "status", "not found", "err", err)
				continue
			}

			from := User{
				ID:   bm.FromID,
				Name: bm.FromName,
			}

			if err := c.sendMessage(from, to, bm.Text); err != nil {
				c.log.Error("listenBUS: sending message failed", "err", err)
			}

			c.log.Info("listenBUS: sent message", "from", bm.FromID, "to", to.ID)
		}
	}()
}

func (c *Chat) readMessageFromBUS(ctx context.Context) (jetstream.Msg, error) {
	type response struct {
		msg jetstream.Msg
		err error
	}

	ch := make(chan response, 1)

	go func() {
		msg, err := c.consumer.Next()
		if err != nil {
			ch <- response{err: fmt.Errorf("nextMsgWithContext: %w", err)}
		} else {
			ch <- response{msg: msg}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
		//no ack
	case resp := <-ch:
		if resp.err != nil {
			return nil, resp.err
		}
		//ack the message
		if err := resp.msg.Ack(); err != nil {
			return nil, fmt.Errorf("ack message: %w", err)
		}
		return resp.msg, nil
	}
}
