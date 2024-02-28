package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/zekroTJA/timedmap"
)

// In order for the modal submission to be useful, the context for it's initial request must be stored.
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
			// An option was focused, but it does not have a handler.
			var focusedOption *discordgo.ApplicationCommandInteractionDataOption
			focusedIndex := 0
			for i, option := range data.Options {
				if option.Focused {
					focusedOption = option
					focusedIndex = i
					break
				}
			}
			log.WithFields(logrus.Fields{"focusedIndex": focusedIndex, "focusedOption": focusedOption.Name, "focusedOption.value": focusedOption.Value}).Warn("Unhandled autocomplete option")
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

var (
	LocationOption = &discordgo.ApplicationCommandOption{
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         "location",
		Description:  "The complex to register with",
		Required:     true,
		Autocomplete: true,
	}
	GuestCodeOption = &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "code",
		Description: "The guest code, if required",
		Required:    false,
	}
	RegisterCommandDefinition = &discordgo.ApplicationCommand{
		Name:        "register",
		Description: "Register a vehicle for parking",
		Options: []*discordgo.ApplicationCommandOption{
			LocationOption,
			GuestCodeOption,
		},
	}
)

func RegisterCommandHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	log := logrus.WithFields(logrus.Fields{
		"interaction": interaction.ID,
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

		var useStoredCode bool
		var code string

		// Check if a guest code was provided (and set it)
		_, guestCodeProvided := lo.Find(data.Options, func(option *discordgo.ApplicationCommandInteractionDataOption) bool {
			code = option.StringValue()
			return option.Name == GuestCodeOption.Name
		})

		userId, parseErr := strconv.Atoi(interaction.Member.User.ID)
		if parseErr != nil {
			HandleError(session, interaction, parseErr, "Error occurred while parsing user id")
			return
		}

		// Check if a guest code is required for this location
		guestCodeCondition := GetCodeRequirement(int64(location_id))

		// TODO: Add case for when guest code is provided but not required

		// Circumstance under which error is certain
		if !guestCodeProvided && guestCodeCondition == GuestCodeNotRequired {
			// A guest code could be stored, so check for it.
			log.WithField("location", location_id).Debug("No guest code provided for location, but one is not required. Checking for stored code.")
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

		// Get the form for the location
		var form GetFormResult
		if guestCodeProvided {
			form = GetVipForm(uint(location_id), code)

			// requireGuestCode being returned for a VIP form indicates an invalid code.
			if form.requireGuestCode {
				// Handling is the same for both cases, but the message differs & the code is removed if it was stored.
				if useStoredCode {
					HandleError(session, interaction, nil, ":x: This location requires a guest code and the one stored was not valid (and subsequently deleted).")
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
		registrationFormComponents := FormToModalComponents(form)

		// The ID of the original interaction is used as the identifier for the registration context (uint64)
		registerIdentifier, parseErr := strconv.ParseUint(interaction.ID, 10, 64)
		if parseErr != nil {
			HandleError(session, interaction, parseErr, "Error occurred while parsing interaction message identifier")
		}

		// Parse resident profile ID
		residentProfileId, parseErr := strconv.ParseUint(form.residentProfileId, 10, 64)
		if parseErr != nil {
			HandleError(session, interaction, parseErr, "Error occurred while parsing resident profile identifier")
		}

		// Log the registration context at debug
		log.WithFields(logrus.Fields{
			"registerIdentifier": registerIdentifier,
			"propertyId":         location_id,
			"residentId":         form.residentProfileId,
		})

		// Store the registration context for later use
		SubmissionContexts.Set(registerIdentifier, &RegisterContext{
			hiddenKeys: form.hiddenInputs,
			propertyId: uint(location_id),
			requiredFormKeys: lo.Map(form.fields, func(field Field, _ int) string {
				return field.id
			}),
			residentId: uint(residentProfileId),
		}, time.Hour)

		registrationFormComponents = append(registrationFormComponents, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:  "email",
					Label:     "Email Address (for confirmation)",
					Style:     discordgo.TextInputShort,
					Required:  false,
					MinLength: 1,
				},
			}})

		response := discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID:   "register:" + interaction.ID,
				Title:      "Vehicle Registration",
				Components: registrationFormComponents,
			},
		}

		err := session.InteractionRespond(interaction.Interaction, &response)
		if err != nil {
			log.WithField("dump", spew.Sdump(response)).Error(err)
		}

	// Autocomplete is used to provide the user with a list of locations to choose from
	case discordgo.InteractionApplicationCommandAutocomplete:
		data := interaction.ApplicationCommandData()
		var choices []*discordgo.ApplicationCommandOptionChoice

		// Find the focused option
		focusedOption, _ := lo.Find(data.Options, func(option *discordgo.ApplicationCommandInteractionDataOption) bool {
			return option.Focused
		})

		switch focusedOption.Name {
		case LocationOption.Name:
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
			// An option was focused, but it does not have a handler.
			log.WithFields(logrus.Fields{"focusedOption": focusedOption.Name, "focusedOption.value": focusedOption.Value}).Warn("Unhandled autocomplete option")
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
