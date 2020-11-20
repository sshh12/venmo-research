package storage

// User is a postgres user
type User struct {
	// Meta
	tableName struct{} `pg:",discard_unknown_columns"`

	// Venmo Fields
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

	// Research Fields
	BingResults     map[string]interface{} `pg:"type:'json'"`
	DDGResults      string                 `pg:"type:'text'"`
	FacebookResults map[string]interface{} `pg:"type:'json'"`
	FacebookProfile map[string]interface{} `pg:"type:'json'"`
	PeekYouResults  map[string]interface{} `pg:"type:'json'"`
}

// Transaction is postgres transaction
type Transaction struct {
	ID          int
	Message     string `pg:"type:'varchar'"`
	Story       string `pg:"type:'varchar'"`
	Type        string `pg:"type:'varchar'"`
	Created     string `pg:"type:'timestamp'"`
	Updated     string `pg:"type:'timestamp'"`
	ActorUserID int
	RecipientID int
}

// UserToTransaction is relation between users and transactions
type UserToTransaction struct {
	UserID        int
	TransactionID int
	IsActor       bool `pg:"type:'boolean'"`
}
