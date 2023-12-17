package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
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

		location_id, _ := strconv.Atoi(data.Options[0].StringValue())
		code := data.Options[1].StringValue()
		user_id, _ := strconv.Atoi(interaction.Member.User.ID)

		// TODO: Validate that the location exists
		// TODO: Validate that the code has no invalid characters
		already_set := StoreCode(code, int64(location_id), user_id)
		responseText := "Your guest code at \"%s\" has been set."
		if already_set {
			responseText = "Your guest code at \"%s\" has been updated."
		}

		location := cachedLocationsMap[uint(location_id)]

		session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Footer: &discordgo.MessageEmbedFooter{
							Text: GetFooterText(),
						},
						Description: fmt.Sprintf(responseText, location.name),
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
			log.Printf("Warning: Unhandled autocomplete option: %v", data.Options)
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
			Name:        "code",
			Description: "The guest code, if required",
			Required:    false,
		},
	},
}

func RegisterCommandHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	switch interaction.Type {

	case discordgo.InteractionApplicationCommand:
		location_id, parse_err := strconv.Atoi(interaction.ApplicationCommandData().Options[0].StringValue())
		if parse_err != nil {
			panic(parse_err)
		}

		form := GetForm(uint(location_id))
		if form.err != nil {
			panic(form.err)
		}

		if form.requireGuestCode {
			session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Footer: &discordgo.MessageEmbedFooter{
								Text: GetFooterText(),
							},
							Description: "This location requires a guest code to register a vehicle.",
						},
					},
				},
			})
			return
		}

		registrationFormComponents := FormToComponents(form)

		err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID:   "registration_" + interaction.Interaction.Member.User.ID,
				Title:      "Vehicle Registration",
				Components: registrationFormComponents,
			},
		})
		if err != nil {
			panic(err)
		}

		// TODO: Validate license plate
		// session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
		// 	Data: &discordgo.InteractionResponseData{
		// 		Embeds: []*discordgo.MessageEmbed{
		// 			{
		// 				Footer: &discordgo.MessageEmbedFooter{
		// 					Text: GetFooterText(),
		// 				},
		// 				Description: "testing 123",
		// 				Fields:      []*discordgo.MessageEmbedField{},
		// 			},
		// 		},
		// 		AllowedMentions: &discordgo.MessageAllowedMentions{},
		// 	},
		// })

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
