package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/zekroTJA/timedmap"
)

var SubmissionContexts = timedmap.New(5 * time.Minute)

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
	log := logrus.WithFields(logrus.Fields{
		"interaction": interaction.ID,
		"message":     interaction.Message.Reference(),
		"user":        interaction.Member.User.ID,
		"command":     "code",
	})

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
			focusedIndex := 0
			for i, option := range data.Options {
				if option.Focused {
					focusedIndex = i
					break
				}
			}
			log.WithFields(logrus.Fields{"focusedIndex": focusedIndex}).Warn("Unhandled autocomplete option")
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
	log := logrus.WithFields(logrus.Fields{
		"interaction": interaction.ID,
		"message":     interaction.Message.Reference(),
		"user":        interaction.Member.User.ID,
		"command":     "code",
	})

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
		userId, parseErr := strconv.Atoi(interaction.Member.User.ID)
		if parseErr != nil {
			HandleError(session, interaction, parseErr, "Error occurred while parsing user id")
			return
		}

		// Check if a guest code is required for this location
		guestCodeCondition := GetCodeRequirement(int64(location_id))

		// TODO: Add case for when guest code is provided but not required

		// Circumstane under which error is certain
		if !guestCodeProvided && guestCodeCondition == GuestCodeNotRequired {
			// A guest code could be stored, so check for it.
			log.Debugf("No guest code provided for location %d, but one is required. Checking for stored code.", location_id)
			code = GetCode(int64(location_id), int(userId))

			if code == "" {
				// No code was stored, error out.
				HandleError(session, interaction, nil, ":x: This location requires a guest code.")
				return
			} else {
				// Code available, use it.
				log.WithFields(logrus.Fields{
					"location_id": location_id,
					"code":        code,
				}).Debug("Using stored code for location")
				guestCodeProvided = true
				useStoredCode = true
			}
		}

		if guestCodeProvided {
			form = GetVipForm(uint(location_id), code)

			// requireGuestCode being returned for a VIP form indicates an invalid code.
			if form.requireGuestCode {
				// Handling is the same for both cases, but the message differs & the code is removed if it was stored.
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
				// The code ended up being required, so we mark it as such.
				if guestCodeCondition == Unknown {
					log.WithFields(logrus.Fields{
						"location_id": location_id,
					}).Debug("Marking location as requiring a guest code")
					SetCodeRequirement(int64(location_id), true)
				}
				HandleError(session, interaction, nil, ":x: This location requires a guest code.")
				return
			}
		}

		// Convert the form into message components for a modal presented to the user
		registrationFormComponents := FormToComponents(form)

		// The message ID of the original interaction is used as the identifier for the registration context (uin664)
		registerIdentifier, parseErr := strconv.ParseUint(interaction.Message.ID, 10, 64)
		if parseErr != nil {
			HandleError(session, interaction, parseErr, "Error occurred while parsing interaction message identifier")
		}

		requiredFormKeys := make([]string, len(form.fields))
		for i, field := range form.fields {
			requiredFormKeys[i] = field.id
		}

		// Store the registration context for later use
		SubmissionContexts.Set(registerIdentifier, &RegisterContext{
			hiddenKeys:       form.hiddenInputs,
			propertyId:       uint(location_id),
			requiredFormKeys: requiredFormKeys,
			residentId:       0,
		}, time.Hour)

		err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID:   "register:" + interaction.Message.ID,
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
