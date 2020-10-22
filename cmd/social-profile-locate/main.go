package main

import (
	"fmt"
	"log"

	"github.com/sshh12/venmo-research/storage"
)

var defaultPicURLs = []string{
	"https://s3.amazonaws.com/venmo/no-image.gif",
}

func main() {
	store, err := storage.NewPostgresStore()
	if err != nil {
		log.Fatal(err)
		return
	}
	users, _ := store.SampleUsers(10)
	for _, user := range users {
		processUser(&user)
	}
}

func processUser(user *storage.User) {
	for _, picURL := range defaultPicURLs {
		if picURL == user.PictureURL {
			return
		}
	}
	fmt.Println(user.Name, user.PictureURL)
	// TODO
}
