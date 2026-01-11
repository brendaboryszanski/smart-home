# Alexa Integration

Control your smart home with voice commands through Alexa.

## How It Works

```
"Alexa, tell home to turn on the lights"
         ↓
    Alexa Skill
         ↓
  Cloudflare Tunnel (free HTTPS)
         ↓
    Raspberry Pi (your app)
         ↓
    Home Assistant
         ↓
    Your devices
```

## Setup

See [SETUP.md](SETUP.md) for step-by-step instructions.

**Time required**: ~20-30 minutes
**Cost**: Free (no domain required)

## Example Commands

- "Alexa, tell home to turn on the living room light"
- "Alexa, tell home to turn off all lights"
- "Alexa, tell home to activate movie scene"
- "Alexa, tell home to set bedroom to 50 percent"

## Testing Without Alexa

```bash
# Test your endpoint directly
curl -X POST "https://YOUR_TUNNEL_URL/alexa?token=YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "turn on living room light"
```

## Alternative: IFTTT (Simpler but slower)

If you prefer not to create an Alexa skill, you can use IFTTT:

1. Create account at [ifttt.com](https://ifttt.com)
2. Create applet: **If** Alexa "Say a specific phrase" → **Then** Webhooks "Make web request"
3. Configure webhook to your tunnel URL

**Cons**: 2-5 second delay, requires saying "trigger" keyword.
