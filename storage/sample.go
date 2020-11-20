package storage

// SampleUsersWithoutDDGResults samples users
func (store *Store) SampleUsersWithoutDDGResults(n int) ([]User, error) {
	return store.sampleUsers("SELECT * FROM users WHERE ddg_results is null", n)
}

// SampleUsersWithoutBingResults samples users
func (store *Store) SampleUsersWithoutBingResults(n int) ([]User, error) {
	return store.sampleUsers("SELECT * FROM users WHERE bing_results is null", n)
}

// SampleUsersWithoutFacebookResults samples users
func (store *Store) SampleUsersWithoutFacebookResults(n int) ([]User, error) {
	return store.sampleUsers("SELECT * FROM users WHERE facebook_results is null and picture_url LIKE '%%facebook=true'", n)
}

// SampleUsersWithoutPeekYouResults samples users
func (store *Store) SampleUsersWithoutPeekYouResults(n int) ([]User, error) {
	return store.sampleUsers("SELECT * FROM users WHERE peek_you_results is null and picture_url LIKE '%%facebook=true'", n)
}

// SampleUsersWithFacebookResultsWithoutProfile samples users
func (store *Store) SampleUsersWithFacebookResultsWithoutProfile(n int) ([]User, error) {
	return store.sampleUsers("SELECT * FROM users WHERE facebook_results is not null and picture_url LIKE '%%facebook=true' and facebook_profile is null", n)
}

// SampleUsersWithPeekYouMatchWithoutProfile samples users
func (store *Store) SampleUsersWithPeekYouMatchWithoutProfile(n int) ([]User, error) {
	return store.sampleUsers("SELECT * FROM users WHERE peek_you_results is not null and (peek_you_results ->> 'ResultsMatch') != '[]' and facebook_profile is null", n)
}
