package main

import (
  "os"
  "fmt"
  "flag"
  "syscall"
  "strings"
  "bytes"

  "github.com/mattermost/mattermost-server/model"

  "golang.org/x/crypto/ssh/terminal"
)

func main() {
    serverPtr := flag.String("server", "http://localhost:8065", "a server URL")
    loginPtr := flag.String("login", "", "your login")
    passwordPtr := flag.String("password", "-", "when - then read from stdin")
    teamPtr := flag.String("team", "", "team name")
    channelPtr := flag.String("channel", "", "channel name")
    victimPtr := flag.String("victim", "", "victim user")

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

    if *victimPtr == "" {
        fmt.Println("Missing victim user name - will reply to everyone!")
    }

    wsAddress := strings.Replace(*serverPtr, "http", "ws", 1)
    fmt.Printf("WebSocket address '%s'\n\n", wsAddress)

    Client := model.NewAPIv4Client(*serverPtr)

    var password string
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

    var helloMessage bytes.Buffer
    helloMessage.WriteString("Hello")

    var victimId string
    if *victimPtr != "" {
        if victim, resp := Client.GetUserByUsername(*victimPtr, ""); resp.Error != nil {
            fmt.Printf("Cannot find the user '%s'\n\n", *victimPtr)
            PrintServerError(resp.Error)
            os.Exit(1)
        } else {
            victimId = victim.Id
        }

        helloMessage.WriteString(" @")
        helloMessage.WriteString(*victimPtr)
    }

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

    if nil != PostMessageToChannel(Client, channel.Id, "", helloMessage.String()) {
        os.Exit(1)
    }

    fmt.Println("Posted!")

    webSocketClient, err := model.NewWebSocketClient4(wsAddress, Client.AuthToken)
    fmt.Println("Client created ...")

    if err != nil {
        fmt.Printf("Failed to connects WebSocket at '%s'\n\n", wsAddress)
        PrintServerError(err)
        os.Exit(1)
    }

    webSocketClient.Listen()
    fmt.Println("Listening ...")

    go func() {
        for {
            select {
                case resp := <-webSocketClient.EventChannel:
                    HandleWebSocketResponse(Client, channel.Id, victimId, resp)
           }
        }
    }()

    select {}
}


func PrintServerError(err *model.AppError) {
	fmt.Println("\tError Details:")
	fmt.Println("\t\t" + err.Message)
	fmt.Println("\t\t" + err.Id)
	fmt.Println("\t\t" + err.DetailedError)
}

func PostMessageToChannel(client *model.Client4, channelId string, postId string, msg string) *model.AppError {
    post := &model.Post{}
    post.ChannelId = channelId
    post.RootId = postId
    post.Message = msg

    if _, resp := client.CreatePost(post); resp.Error != nil {
        fmt.Println("Failed to post the message")
        PrintServerError(resp.Error)
        return resp.Error
    }

    return nil
}

func HandleWebSocketResponse(client *model.Client4, channelId string, victimId string, event *model.WebSocketEvent) {
    if event.Broadcast.ChannelId != channelId {
        return
    }

    if event.Event != model.WEBSOCKET_EVENT_POSTED {
        return
    }

    post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))
    if post != nil {
        if victimId != "" && post.UserId != victimId {
            return
        }

        if post.Message != "Thank you!" {
            PostMessageToChannel(client, channelId, post.Id, "Thank you!")
        }
    }
}
