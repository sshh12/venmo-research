package storage

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/sshh12/venmo-research/venmo"
)

const flushThreshold = 5000

func init() {
	orm.RegisterTable((*UserToTransaction)(nil))
}

// Store is a storage client
type Store struct {
	db     *pg.DB
	buffer chan interface{}
	mux    sync.Mutex
}

func createTables(db *pg.DB) error {
	models := []interface{}{
		(*User)(nil),
		(*Transaction)(nil),
		(*UserToTransaction)(nil),
	}
	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{IfNotExists: true})
		if err != nil {
			return err
		}
	}
	return nil
}

func env(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultVal
}

// NewPostgresStore creates a postgres client
func NewPostgresStore() (*Store, error) {
	opts := &pg.Options{
		User:     env("POSTGRES_USER", "postgres"),
		Password: env("POSTGRES_PASS", "password"),
		Addr:     env("POSTGRES_ADDR", "localhost:5432"),
		Database: env("POSTGRES_DB", "venmo"),
	}
	db := pg.Connect(opts)
	if err := createTables(db); err != nil {
		return nil, err
	}
	log.Printf("Connected to postgres://%s:%s@%s/%s", opts.User, opts.Password, opts.Addr, opts.Database)
	buf := make(chan interface{}, flushThreshold*10)
	return &Store{db: db, buffer: buf}, nil
}

func convertVenmoUserToModel(user *venmo.User) *User {
	ID, _ := strconv.Atoi(user.ID)
	return &User{
		ID:         ID,
		Username:   user.Username,
		PictureURL: user.PictureURL,
		Name:       user.Name,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Created:    user.Created,
		IsBusiness: user.IsBusiness,
		Cancelled:  user.Cancelled,
		ExternalID: user.ExternalID,
	}
}

// AddTransactions adds a transaction to the db
func (store *Store) AddTransactions(item *venmo.FeedItem) error {
	actorModel := convertVenmoUserToModel(&item.Actor)
	store.buffer <- actorModel
	if len(item.Transactions) > 10 {
		panic("Item has more than 10 transactions, oof")
	}
	for idx, trans := range item.Transactions {
		user, err := venmo.CastTargetToUser(trans.Target)
		if err != nil {
			log.Println(trans, err)
			continue
		}
		userModel := convertVenmoUserToModel(user)
		customID := item.PaymentID*10 + idx
		transModel := &Transaction{
			ID:          customID,
			Message:     item.Message,
			Story:       item.StoryID,
			Type:        item.Type,
			Created:     item.Created,
			Updated:     item.Updated,
			ActorUserID: actorModel.ID,
			RecipientID: userModel.ID,
		}
		store.buffer <- userModel
		store.buffer <- transModel
		store.buffer <- &UserToTransaction{
			UserID:        actorModel.ID,
			TransactionID: customID,
			IsActor:       true,
		}
		store.buffer <- &UserToTransaction{
			UserID:        userModel.ID,
			TransactionID: customID,
			IsActor:       false,
		}
	}
	store.mux.Lock()
	if len(store.buffer) >= flushThreshold {
		if err := store.Flush(); err != nil {
			store.mux.Unlock()
			return err
		}
	}
	store.mux.Unlock()
	return nil
}

// Flush flushes the store buffer
func (store *Store) Flush() error {
	users := make([]User, 0)
	transactions := make([]Transaction, 0)
	relations := make([]UserToTransaction, 0)
	loop := true
	for loop {
		select {
		case item := <-store.buffer:
			switch v := item.(type) {
			case *User:
				users = append(users, *v)
			case *Transaction:
				transactions = append(transactions, *v)
			case *UserToTransaction:
				relations = append(relations, *v)
			default:
				panic("Unknown model type")
			}
		default:
			loop = false
		}
	}
	log.Printf("Flushing %d users, %d transactions (%d)", len(users), len(transactions), len(relations))
	_, err := store.db.Model(&users).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	_, err = store.db.Model(&transactions).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	_, err = store.db.Model(&relations).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}

func (store *Store) sampleUsers(query string, n int) ([]User, error) {
	var users []User
	_, err := store.db.Query(&users, query+fmt.Sprintf(" ORDER BY RANDOM() LIMIT %d", n))
	return users, err
}

// UpdateUser updates a user
func (store *Store) UpdateUser(user *User) error {
	_, err := store.db.Model(user).WherePK().Update()
	if err != nil {
		return err
	}
	return nil
}
