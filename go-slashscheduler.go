package slashscheduler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/dougrich/go-discordbot"
	"github.com/hashicorp/go-memdb"
)

type SlashScheduler struct {
	*memdb.MemDB
}

func (SlashScheduler) Name() string {
	return "SlashScheduler"
}

func (scheduler SlashScheduler) Register(bot *discordbot.Bot) error {
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
						Name:        "title",
						Description: "the new title of the schedule",
					},
				},
			},
		},
		"schedule commands",
		func(ctx context.Context, args *discordbot.Arguments) error {
			newtitle := ""
			if err := args.Scan(&newtitle); err != nil {
				return err
			}
			txn := slashSchedulerTxn{scheduler.Txn(newtitle != "")}
			s, err := txn.get(discordbot.GuildID(ctx))
			if err != nil {
				txn.Abort()
				return err
			}
			sv := *s
			if newtitle != "" {
				sv.Title = newtitle
				err = txn.replace(s, sv)
				if err != nil {
					txn.Abort()
					return err
				}
			}

			return bot.Respond(ctx, discordbot.WithMessage(sv.Message()), discordbot.WithEmbed(sv.Embed()))
		},
	)
	return nil
}
