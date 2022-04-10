package slashscheduler

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dougrich/go-discordbot"
)

var (
	discordTemplates = template.FuncMap{
		"mention_time": func(timestamp int64) string {
			return fmt.Sprintf("<t:%d:T>", timestamp)
		},
	}
	templateMessage = scheduleTemplate("templateMessage", "{{if .Enabled }}schedule is **enabled**, next game is at {{mention_time .Timestamp}}{{else}}schedule is **disabled**, requires both a time and a channel id{{end}}")
	templateTagline = scheduleTemplate("templateTagline", "Hey @everyone, reminder that we have an upcoming at {{mention_time .Timestamp}}, see you then!")
)

func scheduleTemplate(name string, t string) *template.Template {
	return template.Must(template.New(name).Funcs(discordTemplates).Parse(t))
}

type Schedule struct {
	GuildID     string
	Title       string
	Link        string
	Description string
	Recurring   bool
	Color       int
	Timestamp   int64
	ChannelID   string
}

func (s Schedule) mustExecute(t *template.Template) string {
	var sb strings.Builder
	err := t.Execute(&sb, s)
	if err != nil {
		panic(fmt.Errorf("slashscheduler: template failed to execute: %v", err.Error()))
	}
	return sb.String()
}

func (s Schedule) Message() string {
	return s.mustExecute(templateMessage)
}

func (s Schedule) Tagline() string {
	return s.mustExecute(templateTagline)
}

func (s Schedule) Enabled() bool {
	return s.Timestamp >= time.Now().Unix() && s.ChannelID != ""
}

func (s Schedule) Embed() *discordgo.MessageEmbed {
	e := discordgo.MessageEmbed{
		Title:       s.Title,
		Description: s.Description,
		Color:       s.Color,
	}

	if e.Title == "" {
		e.Title = "*undefined* - set with `/schedule title newtitle:string`"
	}

	if e.Description == "" {
		e.Description = "*undefined* - set with `/schedule description newdescription:string`"
	}

	if s.Link == "" {
		e.Description = e.Description + "\n\n *link undefined* - set with `/schedule link newlink:string`"
	} else {
		e.Type = discordgo.EmbedTypeLink
		e.URL = s.Link
	}

	if s.Enabled() {
		e.Timestamp = time.Unix(s.Timestamp, 0).Format(time.RFC3339)
	}

	return &e
}

func (s Schedule) Notify(bot *discordbot.Bot) {
	bot.Message(
		s.ChannelID,
		discordbot.WithMessage(s.Tagline()),
		discordbot.WithEmbed(s.Embed()),
		discordbot.WithReaction("👍"),
		discordbot.WithReaction("❌"),
	)
}
