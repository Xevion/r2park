package main

import (
	"strconv"

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
	// email := dataByCustomID["email"].(*discordgo.TextInput).Value

	// Collect the form parameters provided by the user
	formParams := map[string]string{}
	for fieldName, component := range dataByCustomID {
		// Email is a special case, it's not part of the form
		if fieldName == "email" {
			continue
		}

		formParams[fieldName] = component.(*discordgo.TextInput).Value
	}

	// The ID of the original interaction is used as the identifier for the registration context (uint64)
	registerIdentifier, parseErr := strconv.ParseUint(interaction.ID, 10, 64)
	if parseErr != nil {
		HandleError(session, interaction, parseErr, "Error occurred while parsing interaction message identifier")
	}

	// Get the context that we stored prior to emitting the modal
	context := SubmissionContexts.GetValue(registerIdentifier).(*RegisterContext)

	// Register the vehicle
	result, err := RegisterVehicle(formParams, context.propertyId, context.residentId, context.hiddenKeys)

	if err != nil {
		HandleError(session, interaction, err, "Failed to register vehicle")
		return
	}

	log.Infof("RegisterVehicle result: %+v", result)

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
