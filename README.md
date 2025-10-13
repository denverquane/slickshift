<p align="center">
    <a href="https://hub.docker.com/repository/docker/denverquane/slickshift" alt="Pulls">
        <img src="https://img.shields.io/docker/pulls/denverquane/slickshift.svg" />
    </a>
    <a href="https://goreportcard.com/report/github.com/denverquane/slickshift" alt="Report Card">
        <img src="https://goreportcard.com/badge/github.com/denverquane/slickshift" />
    </a>
</p>

<p align="center">
    <a href="https://discord.com/oauth2/authorize?client_id=1420238749270544547&permissions=0&integration_type=0&scope=bot" alt="invite">
        <img alt="Invite Link" src="https://img.shields.io/static/v1?label=bot&message=invite%20me&color=purple">
    </a>
    <a href="https://discord.gg/GDSsKcrPxp" alt="Discord Link">
        <img src="https://img.shields.io/discord/1423423052238159925?logo=discord" />
    </a>
</p>

## SlickShift

Slickshift is a Discord Bot to automatically redeem SHiFT codes for Borderlands 4!

It uses simple HTTP requests to authenticate with [SHiFT Rewards](https://shift.gearboxsoftware.com/rewards), and then store user cookies in order to redeem SHiFT codes on their behalf.

User cookies are encrypted in a sqlite database, and SHiFT codes can be provided at whim using the API server on port `8080`, or via the Discord Slash Command `/add`.

*SlickShift is not affiliated with, endorsed by, or approved by Gearbox Software, 2K Games, or the SHiFT service in any way.* To see more details, see [LIABILITY.md](./LIABILITY.md)

### Installation

Install Go 1.25+, then run `go build -o bot-exec cmd/server.go`. Then run the output `bot-exec` 
executable with the [Environment Variables](#environment-variables) specified below

*TODO: Add Github workflow to generate executables automatically*

### Docker

SlickShift is available as a Docker image in Dockerhub under [denverquane/slickshift](https://hub.docker.com/repository/docker/denverquane/slickshift/general). 

See [Environment Variables](#environment-variables) for required runtime information and config.

### Environment Variables

| Variable             | Required | Default       | Description                                                                                                                                                                  |
|----------------------| -------- |---------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ENCRYPTION_KEY_B64` | ✅ Yes    | *None*        | Base64-encoded 32-byte secret key used to encrypt user data. The program will exit if this is not set or invalid. Generate this with `openssl rand -base64 32`               |
| `DISCORD_BOT_TOKEN`  | ✅ Yes    | *None*        | Discord bot token used to authenticate with the Discord API. The program will exit if this is not set.                                                                       |
| `DISCORD_GUILD_ID`   | ❌ No    | *None*        | The ID of the Discord guild (server) where the bot will operate. If not set, slash commands will be registered globally (not recommended for development).                   |
| `REDEEM_INTERVAL`    | ❌ No     | `30` (minutes) | Interval (in minutes) between redemption attempts. Must be ≥ 1. (Adding codes or registering new users will always trigger the redemption loop, so this can be a high value) |
| `DATABASE_FILE_PATH` | ❌ No     | `./sqlite.db` | Path to the SQLite database file. If not set, it defaults to a local file.                                                                                                   |
| `API_SERVER_PORT`    | ❌ No     | `8080`        | Port that the API server will be accessible on.                                                                                                                              |