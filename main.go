package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	OwnerID     string   `json:"owner_id"`
	GuildID     string   `json:"guild_id"`
	Token       string   `json:"token"`
	BadPhrases  []string `json:"bad_phrases"`
	DeleteAfter string   `json:"delete_after"`
}

type BadMessage struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channel_id"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	AuthorID  string    `json:"author_id"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	config  Config
	sc      = make(chan os.Signal, 1)
	running bool
	totals  = []*BadMessage{}
)

func main() {
	d, err := os.ReadFile("./config.json")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(d, &config)
	s, _ := discordgo.New("Bot " + config.Token)
	s.AddHandler(onReady)
	//s.AddHandler(onMessage)
	//s.AddHandler(onMessage2)
	s.Open()
	defer close(s)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	<-sc
}

func onMessage2(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.Author.ID != config.OwnerID || m.GuildID != config.GuildID || m.Content != "run" || running {
		return
	}
	running = true
	d, err := os.ReadFile("./msgs.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	var bads []*BadMessage
	json.Unmarshal(d, &bads)

	fmt.Println(len(bads))

	deleted := 0

	for i, m := range bads {
		err = s.ChannelMessageDelete(m.ChannelID, m.ID)
		if err != nil {
			fmt.Println(err)
			fmt.Println(fmt.Sprintf("could not delete #%v / %v | content: %v", i, len(bads), m.Content))
			continue
		}
		deleted++
		fmt.Println(fmt.Sprintf("deleted #%v / %v | content: %v", i, len(bads), m.Content))
	}
	fmt.Println("deleted", deleted, "/", len(bads))

}
func close(s *discordgo.Session) {
	fmt.Println("closing session..")
	s.Close()
	fmt.Println("session closed")
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Logged in as", r.User)
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.Author.ID != config.OwnerID || m.GuildID != config.GuildID || m.Content != "run" || running {
		return
	}

	running = true
	fmt.Println("it approaches")
	chs, err := s.GuildChannels(m.GuildID)
	if err != nil {
		fmt.Println(err)
		return
	}

	// this can be done with multiple bots instead of just 1
	for _, ch := range chs {
		if ch.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		fmt.Println("checking channel:", ch.Name)
		erase(s, ch)
	}

	fmt.Println("it has been done")
	sc <- syscall.SIGTERM
}

func erase(s *discordgo.Session, ch *discordgo.Channel) {
	after := config.DeleteAfter

	for i := 0; ; i++ {
		msgs, err := s.ChannelMessages(ch.ID, 100, "", after, "")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("current iteration:", i)

		for _, m := range msgs {
			for _, p := range config.BadPhrases {
				if strings.Contains(strings.ToLower(m.Content), strings.ToLower(p)) {
					totals = append(totals, &BadMessage{m.ID, m.ChannelID, m.Content, m.Author.String(), m.Author.ID, m.Timestamp})
				}
			}
		}

		if len(msgs) != 100 {
			break
		}
		after = msgs[0].ID
	}
}
