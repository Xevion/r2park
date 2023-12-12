package main

import (
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

var CodeCommandDefinition = &discordgo.ApplicationCommand{
	Name:        "code",
	Description: "Set the guest code for a given location",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         "location",
			Description:  "The complex to set the code for",
			Required:     true,
			Autocomplete: true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "code",
			Description: "The new code to set",
			Required:    true,
		},
	},
}

func CodeCommandHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	switch interaction.Type {

	case discordgo.InteractionApplicationCommand:
		data := interaction.ApplicationCommandData()

		location := data.Options[0].IntValue()
		code := data.Options[1].StringValue()
		user_id, _ := strconv.Atoi(interaction.Member.User.ID)

		// TODO: Validate that the location exists
		// TODO: Validate that the code has no invalid characters
		StoreCode(code, location, user_id)

		session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Footer: &discordgo.MessageEmbedFooter{
							Text: GetFooterText(),
						},
						Description: "Your guest code has been registered.",
						Fields:      []*discordgo.MessageEmbedField{},
					},
				},
				AllowedMentions: &discordgo.MessageAllowedMentions{},
			},
		})

	case discordgo.InteractionApplicationCommandAutocomplete:
		data := interaction.ApplicationCommandData()
		var choices []*discordgo.ApplicationCommandOptionChoice

		LocationOption := data.Options[0]

		switch {
		case LocationOption.Focused:
			// Seed value is based on the user ID + a 15 minute interval)
			user_id, _ := strconv.Atoi(interaction.Member.User.ID)
			seed_value := int64(user_id) + (time.Now().Unix() / 15 * 60)
			locations := FilterLocations(GetLocations(), data.Options[0].StringValue(), 25, seed_value)

			// Convert the locations to choices
			choices = make([]*discordgo.ApplicationCommandOptionChoice, len(locations))
			for i, location := range locations {
				choices[i] = &discordgo.ApplicationCommandOptionChoice{
					Name:  location.name,
					Value: strconv.Itoa(int(location.id)),
				}
			}
		default:
			log.Printf("Warning: Unhandled autocomplete option: %s", data.Options)
			return
		}

		err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices,
			},
		})
		if err != nil {
			panic(err)
		}
	}
}

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
			MaxLength:   15,
			Required:    true,
			// TODO: Add autocomplete
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "model",
			Description: "Model of Vehicle (e.g. Civic)",
			MaxLength:   15,
			Required:    true,
			// TODO: Add autocomplete
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "plate",
			Description: "License Plate Number (e.g. 123ABC)",
			MaxLength:   8,
			Required:    true,
		},
	},
}

func RegisterCommandHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	switch interaction.Type {

	case discordgo.InteractionApplicationCommand:
		// TODO: Validate license plate
		session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Footer: &discordgo.MessageEmbedFooter{
							Text: GetFooterText(),
						},
						Description: "testing 123",
						Fields:      []*discordgo.MessageEmbedField{},
					},
				},
				AllowedMentions: &discordgo.MessageAllowedMentions{},
			},
		})

	case discordgo.InteractionApplicationCommandAutocomplete:
		data := interaction.ApplicationCommandData()
		var choices []*discordgo.ApplicationCommandOptionChoice

		LocationOption := data.Options[0]
		MakeOption := data.Options[1]
		ModelOption := data.Options[2]

		switch {
		case LocationOption.Focused:
			// Seed value is based on the user ID + a 15 minute interval)
			user_id, _ := strconv.Atoi(interaction.Member.User.ID)
			seed_value := int64(user_id) + (time.Now().Unix() / 15 * 60)
			locations := FilterLocations(GetLocations(), data.Options[0].StringValue(), 25, seed_value)

			// Convert the locations to choices
			choices = make([]*discordgo.ApplicationCommandOptionChoice, len(locations))
			for i, location := range locations {
				choices[i] = &discordgo.ApplicationCommandOptionChoice{
					Name:  location.name,
					Value: strconv.Itoa(int(location.id)),
				}
			}
		case MakeOption.Focused:

		case ModelOption.Focused:
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
