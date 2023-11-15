package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
        // The file token.json stores the user's access and refresh tokens, and is
        // created automatically when the authorization flow completes for the first
        // time.
        tokFile := "token.json"
        tok, err := tokenFromFile(tokFile)
        if err != nil {
                tok = getTokenFromWeb(config)
                saveToken(tokFile, tok)
        }
        return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
        authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
        fmt.Printf("Go to the following link in your browser then type the "+
                "authorization code: \n%v\n", authURL)

        var authCode string
        if _, err := fmt.Scan(&authCode); err != nil {
                log.Fatalf("Unable to read authorization code: %v", err)
        }

        tok, err := config.Exchange(context.TODO(), authCode)
        if err != nil {
                log.Fatalf("Unable to retrieve token from web: %v", err)
        }
        return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
        f, err := os.Open(file)
        if err != nil {
                return nil, err
        }
        defer f.Close()
        tok := &oauth2.Token{}
        err = json.NewDecoder(f).Decode(tok)
        return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
        fmt.Printf("Saving credential file to: %s\n", path)
        f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
        if err != nil {
                log.Fatalf("Unable to cache oauth token: %v", err)
        }
        defer f.Close()
        json.NewEncoder(f).Encode(token)
}

type Email struct {
        Date int64
        Id string
        Body string
        Subject string
        From string
}

func main() {
        ctx := context.Background()
        b, err := os.ReadFile("credentials.json")
        if err != nil {
                log.Fatalf("Unable to read client secret file: %v", err)
        }

        // If modifying these scopes, delete your ipreviously saved token.json.
        config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
        if err != nil {
                log.Fatalf("Unable to parse client secret file to config: %v", err)
        }
        client := getClient(config)

        srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
        if err != nil {
                log.Fatalf("Unable to retrieve Gmail client: %v", err)
        }
        
        // get all messages that match the given query
        currTime := time.Now()
        date := fmt.Sprintf("%d/%d/%d", currTime.Year(), currTime.Month(), currTime.Day() - 1)
        query := fmt.Sprintf("subject:(Thank you for applying) AND after:%s", date)

        user := "me"
        r, err := srv.Users.Messages.List(user).Q(query).Do()
        if err != nil {
                log.Fatalf("Unable to retrieve labels: %v", err)
        }
        if len(r.Messages) == 0 {
                fmt.Println("No messages found.")
                return
        }
        
        // get more details of each email, collect for processing
        emails := []Email{}
        fmt.Println("Messages:")
        for _, m := range r.Messages {
                fmt.Printf("Found message ID: %s\n", m.Id)  
                currEmail := Email{}
                msg, err := srv.Users.Messages.Get(user, m.Id).Do()
                if err != nil {
                        log.Fatalf("Unable to retreive message of ID %s: %v", m.Id, err)
                }
                
                // Decode the raw format of the message. Encoded in base64url in RFC 2822 format
                //decS, _ := base64.URLEncoding.DecodeString(msg.Raw)
                //currEmail.Raw = string(decS)
                currEmail.Id = msg.Id
                currEmail.Date = msg.InternalDate

                for _, h := range msg.Payload.Headers {
                        if h.Name == "From" {
                                currEmail.From = h.Value
                        }
                        
                        if h.Name == "Subject" {
                                currEmail.Subject = h.Value
                        }
                }

                for _, part := range msg.Payload.Parts {
                        if part.MimeType == "text/plain" {
                                encBody := part.Body.Data
                                decBody, _ := base64.URLEncoding.DecodeString(encBody)
                                currEmail.Body = string(decBody)
                        }
                }

                emails = append(emails, currEmail)

                fmt.Printf("Email body: %s\n", currEmail.Body)
        }

        // 
}
