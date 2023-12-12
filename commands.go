package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

var RegisterCommandDefinition = &discordgo.ApplicationCommand{
	Name:        "register",
	Description: "Register a vehicle for parking",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         "location",
			Description:  "The complex to register with",
			Required:     true,
			Autocomplete: true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "make",
			Description: "Make of Vehicle (e.g. Honda)",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "model",
			Description: "Model of Vehicle (e.g. Civic)",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "plate",
			Description: "License Plate Number (e.g. 123ABC)",
			Required:    true,
		},
	},
}

func RegisterCommandHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	switch interaction.Type {
	case discordgo.InteractionApplicationCommand:
		session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Footer: &discordgo.MessageEmbedFooter{
							Text: fmt.Sprintf("Fetched at %s", time.Now().Format("Monday, January 2, 2006 at 3:04:05PM")),
						},
						Description: "",
						Fields:      []*discordgo.MessageEmbedField{},
					},
				},
				AllowedMentions: &discordgo.MessageAllowedMentions{},
			},
		})
	case discordgo.InteractionApplicationCommandAutocomplete:
		data := interaction.ApplicationCommandData()

		var choices []*discordgo.ApplicationCommandOptionChoice

		if data.Options[0].Focused {
			// Seed value is based on the user ID + a 15 minute interval)
			user_id, _ := strconv.Atoi(interaction.Member.User.ID)
			seed_value := int64(user_id) + (time.Now().Unix() / 15 * 60)
			locations := FilterLocations(GetLocations(), data.Options[0].StringValue(), 25, seed_value)

			choices = make([]*discordgo.ApplicationCommandOptionChoice, len(locations))
			for i, location := range locations {
				choices[i] = &discordgo.ApplicationCommandOptionChoice{
					Name:  location.name,
					Value: strconv.Itoa(int(location.id)),
				}
			}
		} else {
			choices = []*discordgo.ApplicationCommandOptionChoice{}
		}

		err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices, // This is basically the whole purpose of autocomplete interaction - return custom options to the user.
			},
		})
		if err != nil {
			panic(err)
		}
	}
}