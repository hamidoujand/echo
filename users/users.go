package users

import (
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/echo/chat"
)

type Users struct {
	log       *slog.Logger
	users     map[uuid.UUID]chat.User
	usersLock sync.RWMutex
}

func New(log *slog.Logger) *Users {
	return &Users{
		users: make(map[uuid.UUID]chat.User),
		log:   log,
	}
}

func (u *Users) Add(usr chat.User) error {
	u.usersLock.Lock()
	defer u.usersLock.Unlock()

	if _, ok := u.users[usr.ID]; ok {
		return chat.ErrUserAlreadyExists
	}

	u.users[usr.ID] = usr
	u.log.Info("added user to the connection map", "id", usr.ID, "name", usr.Name)
	return nil
}

func (u *Users) Retrieve(userID uuid.UUID) (chat.User, error) {
	u.usersLock.RLock()
	defer u.usersLock.RUnlock()
	usr, ok := u.users[userID]
	if !ok {
		return chat.User{}, chat.ErrUserNotFound
	}

	return usr, nil
}

func (u *Users) Connections() map[uuid.UUID]chat.Connection {
	u.usersLock.RLock()
	defer u.usersLock.RUnlock()

	result := make(map[uuid.UUID]chat.Connection)
	for id, usr := range u.users {
		c := chat.Connection{
			Conn:     usr.Conn,
			LastPong: usr.LastPong,
			LastPing: usr.LastPing,
		}

		result[id] = c
	}

	return result
}

func (u *Users) Remove(userID uuid.UUID) {
	u.usersLock.Lock()
	defer u.usersLock.Unlock()

	usr, ok := u.users[userID]
	if !ok {
		u.log.Info("removing user failed, user not found", "id", usr.ID, "name", usr.Name)
		return
	}

	delete(u.users, userID)
	u.log.Info("removing user", "id", usr.ID, "name", usr.Name)
}

func (u *Users) UpdateLastPong(usrID uuid.UUID) (chat.User, error) {
	u.usersLock.Lock()
	defer u.usersLock.Unlock()

	usr, exists := u.users[usrID]
	if !exists {
		return chat.User{}, chat.ErrUserNotFound
	}
	usr.LastPong = time.Now()
	u.users[usrID] = usr
	return usr, nil
}

func (u *Users) UpdateLastPing(usrID uuid.UUID) error {
	u.usersLock.Lock()
	defer u.usersLock.Unlock()

	usr, exists := u.users[usrID]
	if !exists {
		return chat.ErrUserNotFound
	}
	usr.LastPing = time.Now()
	u.users[usrID] = usr
	return nil
}
