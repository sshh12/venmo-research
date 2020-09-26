package storage

import (
	"fmt"

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
}

// Transaction is postgres transaction
type Transaction struct {
	ID int
}

// UserToTransaction is relation between users and transactions
type UserToTransaction struct {
	UserID        int
	TransactionID int
}

// Store is a storage client
type Store struct {
	db *pg.DB
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
		Password: "",
		Database: "venmo",
	})
	// defer db.Close()
	if err := createTables(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil

	// values := []interface{}{
	// 	&User{Id: 1},
	// 	&User{Id: 2},
	// 	&Transaction{Id: 1},
	// 	&UserToTransaction{UserId: 1, TransactionId: 1},
	// 	&UserToTransaction{UserId: 1, TransactionId: 2},
	// }
	// for _, v := range values {
	// 	_, err := db.Model(v).OnConflict("DO NOTHING").Insert()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	// fmt.Println("done")
}

// AddTransaction adds a transaction to the db
func (store *Store) AddTransaction(tran *venmo.FeedItem) {
	fmt.Println(tran)
}
