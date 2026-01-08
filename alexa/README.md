# Alexa Integration

There are 3 ways to connect Alexa with your Smart Home Assistant:

## Option 1: IFTTT (easiest, 5 minutes)

**Pros**: No code, quick setup  
**Cons**: 2-5 second delay, limit of 2 free applets

### Steps:

1. Create account at [ifttt.com](https://ifttt.com)
2. Click "Create" → "If This" → search "Alexa" → "Say a specific phrase"
3. Enter: `amor $` (the $ captures what you say after)
4. "Then That" → search "Webhooks" → "Make a web request"
5. Configure:
   - URL: `https://YOUR_DOMAIN/alexa?token=YOUR_AUTH_TOKEN`
   - Method: `POST`
   - Content Type: `text/plain`
   - Body: `{{TextField}}`
6. Save

**Usage**: "Alexa, trigger amor turn on living room light"

**Note**: You can't use your custom invocation name here, IFTTT requires "trigger" keyword.

---

## Option 2: Custom Skill + Lambda (recommended) 🏆

**Pros**: No delay, more natural invocation ("Alexa, tell home...")  
**Cons**: Requires AWS account (free tier) and more setup

See [SETUP.md](SETUP.md) for detailed step-by-step instructions.

### Architecture:
```
Alexa → AWS Lambda → Cloudflare Tunnel → Raspberry Pi
         (free)        (free)             (your home)
```

**Usage**: "Alexa, amor turn on the living room light"

---

## Option 3: Alexa Routines (Simple Commands Only)

**Pros**: Very natural voice commands, no wake word needed
**Cons**: Each command must be configured manually, no Claude intelligence

If you only need a few fixed commands, use Alexa Routines with your RPI endpoint:

1. In Alexa app → More → Routines → Create Routine
2. When: Voice → Enter phrase: "buenas noches"
3. Add Action → Custom → Enter URL:
   ```
   https://home.yourdomain.xyz/alexa?token=YOUR_TOKEN
   ```
   Method: POST, Body: `activa escena noche`
4. Save

**Usage**: "Alexa, buenas noches" (directly, no "amor" needed)

**Note**: This works with your RPI endpoint but bypasses Claude intelligence. Best for simple, fixed commands like scenes.

---

## Comparison

| Method | Difficulty | Delay | Cost | Natural Voice | Claude AI |
|--------|-----------|-------|------|---------------|-----------|
| IFTTT | Easy | 2-5s | Free (limits) | No ("trigger" needed) | ✅ Yes |
| Skill + Lambda | Medium | <1s | Free | Yes ("amor ...") | ✅ Yes |
| Routines | Easy | <1s | Free | ✅ Very natural | ❌ No (fixed) |

---

## Testing without Alexa

```bash
# Test the endpoint directly (with auth token)
curl -X POST "https://home.yourdomain.com/alexa?token=YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "prende la luz del living"

# Or using header
curl -X POST https://home.yourdomain.com/alexa \
  -H "X-Auth-Token: YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "prende la luz del living"

# Expected response:
# {"status":"ok","message":"Comando recibido"}
```
