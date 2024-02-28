package main

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

func RegisterModalHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	// TODO: Pull in all parameters from the form
	// TODO: Pull in all hidden parameters form database
	// TODO: Pull in resident ID from database

	err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Registration data received. Please wait while your vehicle is registered.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		panic(err)
	}

	data := interaction.ModalSubmitData()
	dataByCustomID := lo.SliceToMap(data.Components, func(c discordgo.MessageComponent) (string, discordgo.MessageComponent) {
		inner := c.(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput)
		return inner.CustomID, inner
	})

	log.Infof("dataByCustomID: %+v", dataByCustomID)

	// Collect the form parameters provided by the user
	formParams := map[string]string{}
	for fieldName, component := range dataByCustomID {
		// Email is a special case, it's not part of the form
		if fieldName == "email" {
			continue
		}

		formParams[fieldName] = component.(*discordgo.TextInput).Value
	}

	// The custom ID of the interaction response is the original identifier (register:\d+)
	_, identifier, _ := strings.Cut(data.CustomID, ":")
	originalIdentifier, parseErr := strconv.ParseUint(identifier, 10, 64)
	if parseErr != nil {
		HandleError(session, interaction, parseErr, "Failed to parse original identifier")
		return
	}

	// Get the contextInterface that we stored prior to emitting the modal
	contextInterface := SubmissionContexts.GetValue(originalIdentifier)
	if contextInterface == nil {
		HandleError(session, interaction, nil, "Failed to retrieve registration context")
		return
	}
	context := contextInterface.(*RegisterContext)

	// Register the vehicle
	result, err := RegisterVehicle(formParams, context.propertyId, context.residentId, context.hiddenKeys)

	if err != nil {
		HandleError(session, interaction, err, "Failed to register vehicle")
		return
	}

	// Send email confirmation if an email was provided
	email := dataByCustomID["email"].(*discordgo.TextInput).Value
	if email != "" {
		success, err := RegisterEmailConfirmation(email, result.vehicleId, strconv.Itoa(int(context.propertyId)))
		if err != nil {
			HandleError(session, interaction, err, "Failed to send email confirmation")
			return
		}

		if !success {
			HandleError(session, interaction, nil, "Failed to send email confirmation")
			return
		}
	}

	// TODO: Edit response to indicate success/failure
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
}
