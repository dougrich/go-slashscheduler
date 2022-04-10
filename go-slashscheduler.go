package slashscheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dougrich/go-discordbot"
	"github.com/hashicorp/go-memdb"
)

var (
	minColor = float64(0)
	maxColor = float64(0xffffff)
)

type SlashScheduler struct {
	*memdb.MemDB
}

func (SlashScheduler) Name() string {
	return "SlashScheduler"
}

func scheduleNotifyPending(bot *discordbot.Bot, txn slashSchedulerTxn, now time.Time) error {
	// go find any games starting in less than 24 hours
	start := now.Add(24 * time.Hour)
	pending, err := txn.pending(now, start)
	if err != nil {
		return err
	}
	for s := range pending {
		if s.Enabled() {
			s.Notify(bot)
			sv := *s
			if s.Recurring {
				start := time.Unix(sv.Timestamp, 0)
				next := start.Add(7 * 24 * time.Hour)
				sv.Timestamp = next.Unix()
			} else {
				sv.Timestamp = 0
			}
			err := txn.replace(s, sv)
			if err != nil {
				return fmt.Errorf("slashscheduler: error occured updating timestamp: %s", err.Error())
			}
		}
	}
	return nil
}

func (scheduler SlashScheduler) Register(bot *discordbot.Bot) error {
	ticker := time.NewTicker(1 * time.Minute)
	bot.Defer(func() {
		ticker.Stop()
	})
	go func() {
		log.Print("slashscheduler: starting notification loop")
		for now := range ticker.C {
			log.Print("slashscheduler: notification loop iteration start")
			start := time.Now()
			txn := slashSchedulerTxn{scheduler.Txn(true)}
			if err := scheduleNotifyPending(bot, txn, now); err != nil {
				log.Printf("slashscheduler: error occured notifying pending: %s", err.Error())
				txn.Abort()
			} else {
				txn.Commit()
			}
			duration := time.Now().Sub(start)
			log.Printf("slashscheduler: notification loop iteration finished, took %.2fs", duration.Seconds())
		}
		log.Print("slashscheduler: notification loop stopped")
	}()

	bot.AddCommand(
		"schedule",
		[]*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "title",
				Description: "get or set the title of the schedule",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "newtitle",
						Description: "the new title of the schedule",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "description",
				Description: "get or set the description of the schedule",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "newdescription",
						Description: "the new description of the schedule",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "link",
				Description: "get or set the link of the schedule",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "newlink",
						Description: "the new link of the schedule",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "timestamp",
				Description: "get or set the starting time of the schedule",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "newtimestamp",
						Description: "the new unix timestamp of the schedule",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "channel",
				Description: "get or set the channel to post the reminder in",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type: discordgo.ApplicationCommandOptionChannel,
						ChannelTypes: []discordgo.ChannelType{
							discordgo.ChannelTypeGuildText,
						},
						Name:        "newchannel",
						Description: "the channel to post the reminder in",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "recurring",
				Description: "get or set if the game is recurring",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type: discordgo.ApplicationCommandOptionInteger,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{
								Name:  "True",
								Value: 1,
							},
							{
								Name:  "False",
								Value: -1,
							},
						},
						Name:        "newrecurring",
						Description: "if the game should be recurring: 1 is true, -1 is false",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "color",
				Description: "get or set the color of the embed",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "newcolor",
						MinValue:    &minColor,
						MaxValue:    maxColor,
						Description: "the color of the embed",
					},
				},
			},
		},
		"schedule commands",
		func(ctx context.Context, args *discordbot.Arguments) error {
			var update func(*Schedule) (string, error)
			switch args.Subcommand() {
			case "title":
				update = func(s *Schedule) (msg string, err error) {
					newtitle := ""
					if err = args.Scan(&newtitle); err == nil && newtitle != "" {
						s.Title = newtitle
						msg = fmt.Sprintf("Title changed to **'%s'**", newtitle)
					} else if s.Title != "" {
						msg = fmt.Sprintf("Title is **'%s'**", s.Title)
					} else {
						msg = "Title is unset"
					}
					return
				}
			case "description":
				update = func(s *Schedule) (msg string, err error) {
					newdescription := ""
					if err = args.Scan(&newdescription); err == nil && newdescription != "" {
						s.Description = newdescription
						msg = fmt.Sprintf("Description changed to\n%s", newdescription)
					} else if s.Description != "" {
						msg = fmt.Sprintf("Description is\n%s", s.Description)
					} else {
						msg = "Description is unset"
					}
					return
				}
			case "link":
				update = func(s *Schedule) (msg string, err error) {
					newlink := ""
					if err = args.Scan(&newlink); err == nil && newlink != "" {
						s.Link = newlink
						msg = fmt.Sprintf("Link changed to '%s'", newlink)
					} else if s.Link != "" {
						msg = fmt.Sprintf("Link is '%s'", s.Link)
					} else {
						msg = "Link is unset"
					}
					return
				}
			case "timestamp":
				update = func(s *Schedule) (msg string, err error) {
					newtimestamp := int64(0)
					if err = args.Scan(&newtimestamp); err == nil && newtimestamp > 0 {
						s.Timestamp = newtimestamp
						msg = fmt.Sprintf("Start time changed to <t:%d:T>", newtimestamp)
					} else if s.Timestamp > 0 {
						msg = fmt.Sprintf("Start time is <t:%d:T>", s.Timestamp)
					} else {
						msg = "Start time is unset"
					}
					return
				}
			case "channel":
				update = func(s *Schedule) (msg string, err error) {
					newchannel := ""
					if err = args.Scan(&newchannel); err == nil && newchannel != "" {
						s.ChannelID = newchannel
						msg = fmt.Sprintf("Channel for notifications changed to <#%s>", newchannel)
					} else if s.ChannelID != "" {
						msg = fmt.Sprintf("Channel for notifications is <#%s>", s.ChannelID)
					} else {
						msg = "Channel is unset"
					}
					return
				}
			case "recurring":
				update = func(s *Schedule) (msg string, err error) {
					newrecurring := int64(0)
					if err = args.Scan(&newrecurring); err == nil && newrecurring != int64(0) {
						s.Recurring = newrecurring == int64(1)
						if s.Recurring {
							msg = "Notifications updated to be recurring (once a week)"
						} else {
							msg = "Notifications updated to stop recurring"
						}
					} else if s.Recurring {
						msg = "Recurring weekly"
					} else {
						msg = "Not recurring"
					}
					return
				}
			case "color":
				update = func(s *Schedule) (msg string, err error) {
					newcolor := int64(-1)
					if err = args.Scan(&newcolor); err == nil && newcolor != -1 {
						msg = fmt.Sprintf("Embed color updated to 0x%X", newcolor)
						s.Color = int(newcolor)
					} else {
						msg = fmt.Sprintf("Embed color 0x%X", s.Color)
					}
					return
				}
			default:
				return errors.New("Unmatched subcommand")
			}
			txn := slashSchedulerTxn{scheduler.Txn(true)}
			s, err := txn.get(discordbot.GuildID(ctx))
			if err != nil {
				txn.Abort()
				return err
			}
			sv := *s
			msg, err := update(&sv)
			if err != nil {
				txn.Abort()
				return err
			}
			err = txn.replace(s, sv)
			if err != nil {
				txn.Abort()
				return err
			}
			txn.Commit()

			return bot.Respond(ctx, discordbot.WithMessage("%s\n%s", msg, sv.Message()), discordbot.WithEmbed(sv.Embed()))
		},
	)
	return nil
}
