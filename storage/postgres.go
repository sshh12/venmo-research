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

func init() {
	orm.RegisterTable((*UserToTransaction)(nil))
}

// User is a postgres user
type User struct {
	ID           int
	Transactions []Transaction `pg:"many2many:user_to_transactions"`
	Username     string
	PictureURL   string
	Name         string
	FirstName    string
	LastName     string
	Created      string
	IsBusiness   bool
	Cancelled    bool
	ExternalID   string
}

// Transaction is postgres transaction
type Transaction struct {
	ID  int
	Msg string
}

// UserToTransaction is relation between users and transactions
type UserToTransaction struct {
	UserID        int
	TransactionID int
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

// NewPostgresStore creates a postgres client
func NewPostgresStore() (*Store, error) {
	db := pg.Connect(&pg.Options{
		User:     "postgres",
		Password: os.Getenv("POSTGRES_PASS"),
		Database: "venmo",
	})
	// defer db.Close()
	if err := createTables(db); err != nil {
		return nil, err
	}
	buf := make(chan interface{}, 1000)
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
	// fmt.Println(item.Message)
	actorModel := convertVenmoUserToModel(&item.Actor)
	// fmt.Println(item.Message, item.PaymentID, item.StoryID)
	store.buffer <- actorModel
	// fmt.Println(actorModel)
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
		transModel := &Transaction{ID: item.PaymentID*10 + idx, Msg: item.Message}
		store.buffer <- userModel
		store.buffer <- transModel
	}
	store.mux.Lock()
	if len(store.buffer) >= 500 {
		store.Flush()
	}
	store.mux.Unlock()
	return nil
}

// Flush flushes the store buffer
func (store *Store) Flush() error {
	fmt.Println("flush")
	len := len(store.buffer)
	values := make([]interface{}, len)
	for i := 0; i < len; i++ {
		values[i] = <-store.buffer
	}
	for _, v := range values {
		_, err := store.db.Model(v).OnConflict("DO NOTHING").Insert()
		if err != nil {
			return err
		}
	}
	return nil
}
