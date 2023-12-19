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
		data := interaction.ApplicationCommandData()

		location_id, parse_err := strconv.Atoi(data.Options[0].StringValue())
		if parse_err != nil {
			HandleError(session, interaction, parse_err, "Error occurred while parsing location id")
			return
		}

		var form GetFormResult
		var code string
		var useStoredCode bool

		guestCodeProvided := len(data.Options) > 1
		if guestCodeProvided {
			code = data.Options[1].StringValue()
		}
		userId, _ := strconv.Atoi(interaction.Member.User.ID)
		guestCodeCondition := GetCodeRequired(int64(location_id))

		// Certain error condition
		if !guestCodeProvided && guestCodeCondition == GuestCodeNotRequired {
			log.Debugf("No guest code provided for location %d, but one is required. Checking for stored code.", location_id)
			code = GetCode(int64(location_id), int(userId))

			if code == "" {
				HandleError(session, interaction, nil, ":x: This location requires a guest code.")
				return
			} else {
				log.Debugf("Using stored code for location %d: %s", location_id, code)
				guestCodeProvided = true
				useStoredCode = true
			}
		}

		if guestCodeProvided {
			form = GetVipForm(uint(location_id), code)
			if form.requireGuestCode {
				if useStoredCode {
					HandleError(session, interaction, nil, ":x: This location requires a guest code and the one stored was not valid & deleted.")
					RemoveCode(int64(location_id), int(userId))
				} else {
					HandleError(session, interaction, nil, ":x: This location requires a guest code and the one provided was not valid.")
				}
				return
			}
		} else {
			form = GetForm(uint(location_id))
			if form.requireGuestCode {
				// Apparently the code was required, so we mark it as such.
				if guestCodeCondition == Unknown {
					log.Debugf("Marking location %d as requiring a guest code", location_id)
					SetCodeRequired(int64(location_id), true)
				}
				HandleError(session, interaction, nil, ":x: This location requires a guest code.")
				return
			}
		}

		registrationFormComponents := FormToComponents(form)

		err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID:   "register:" + interaction.Interaction.Member.User.ID,
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
