package storage

import (
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
	Username     string        `pg:"type:'varchar'"`
	PictureURL   string        `pg:"type:'varchar'"`
	Name         string        `pg:"type:'varchar'"`
	FirstName    string        `pg:"type:'varchar'"`
	LastName     string        `pg:"type:'varchar'"`
	Created      string        `pg:"type:'timestamp'"`
	IsBusiness   bool          `pg:"type:'boolean',default:false"`
	Cancelled    bool          `pg:"type:'boolean',default:false"`
	ExternalID   string        `pg:"type:'varchar'"`
}

// Transaction is postgres transaction
type Transaction struct {
	ID      int
	Message string `pg:"type:'varchar'"`
	Story   string `pg:"type:'varchar'"`
	Type    string `pg:"type:'varchar'"`
	Created string `pg:"type:'timestamp'"`
	Updated string `pg:"type:'timestamp'"`
}

// UserToTransaction is relation between users and transactions
type UserToTransaction struct {
	UserID        int
	TransactionID int
	IsActor       bool `pg:"type:'boolean'"`
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
	buf := make(chan interface{}, 2000)
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
		customID := item.PaymentID*10 + idx
		transModel := &Transaction{
			ID:      customID,
			Message: item.Message,
			Story:   item.StoryID,
			Type:    item.Type,
			Created: item.Created,
			Updated: item.Updated,
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
	if len(store.buffer) >= 1000 {
		store.Flush()
	}
	store.mux.Unlock()
	return nil
}

// Flush flushes the store buffer
func (store *Store) Flush() error {
	len := len(store.buffer)
	log.Printf("Flushing %d", len)
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
