package main

import (
  "os"
  "fmt"
  "flag"
  "syscall"

  "github.com/mattermost/mattermost-server/model"

  "golang.org/x/crypto/ssh/terminal"
)

func main() {
    serverPtr := flag.String("server", "http://localhost:8065", "a server URL")
    loginPtr := flag.String("login", "", "your login")
    passwordPtr := flag.String("password", "-", "when - then read from stdin")
    teamPtr := flag.String("team", "", "team name")
    channelPtr := flag.String("channel", "", "channel name")

    flag.Parse()

    if *loginPtr == "" {
        fmt.Println("Missing login")
        os.Exit(1)
    }

    if *teamPtr == "" {
        fmt.Println("Missing team name")
        os.Exit(1)
    }

    if *channelPtr == "" {
        fmt.Println("Missing channel name")
        os.Exit(1)
    }

    Client := model.NewAPIv4Client(*serverPtr)

    password := ""
    if *passwordPtr == "-" {
        fmt.Print("Enter password: ")

        if bytePassword, err := terminal.ReadPassword(int(syscall.Stdin)); err != nil {
            fmt.Println("Failed to read password")
            os.Exit(1)
        } else {
            password = string(bytePassword)
            fmt.Println("")
        }
    } else {
        password = *passwordPtr
    }

    if _, resp := Client.Login(*loginPtr, password); resp.Error != nil {
        fmt.Printf("Cannot login as the user '%s'\n\n", *loginPtr)
        PrintServerError(resp.Error)
        os.Exit(1)
    }

    password = ""
    *passwordPtr = ""

    team, resp := Client.GetTeamByName(*teamPtr, "")
    if resp.Error != nil {
        fmt.Printf("Cannot find the team '%s'\n\n", *teamPtr)
        PrintServerError(resp.Error)
        os.Exit(1)
    }

    channel, resp := Client.GetChannelByName(*channelPtr, team.Id, "")
    if resp.Error != nil {
        fmt.Printf("Cannot find the channel '%s'\n\n", *channelPtr)
        PrintServerError(resp.Error)
        os.Exit(1)
    }

    post := &model.Post{}
    post.ChannelId = channel.Id
    post.Message = "Hello, world!"

    if _, resp := Client.CreatePost(post); resp.Error != nil {
        fmt.Println("Failed to post the message")
        PrintServerError(resp.Error)
        os.Exit(1)
    }

    fmt.Println("Posted!")
}

func PrintServerError(err *model.AppError) {
	fmt.Println("\tError Details:")
	fmt.Println("\t\t" + err.Message)
	fmt.Println("\t\t" + err.Id)
	fmt.Println("\t\t" + err.DetailedError)
}
