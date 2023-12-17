# r2park-bot

A Discord bot for easily registering parking at the Register2Park parking system.

- Invoke the registration command on any location.
- If a guest code is required, the bot will error and prompt for it.
- If not required, or if invoked with a guest code, the bot will query R2P for the location's registration fields, such as make/model/plate/aptnum.
- The user will then be prompted with a modal containing the registration form.
- Once submitted, all details will be sent to R2P, the user will be notified of success, and the details entered will be preserved.
- The user can then re-use the same details for any other location, or re-register the same location with the same details.

## Feature Ideas

- Register commands in new guilds automatically
- Cache location data for fast autocomplete lookup
- Responses are not sent in the same channel, but instead as a hidden message.
- No code stored with a location can be re-tried after registering a code later.
- Scanning abilities for validating the form structure of all locations Register2Park moderates.
- Ability to forget details registered with a location once, or always for a user.