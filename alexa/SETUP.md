# Alexa Skill Setup

This guide shows how to create an Alexa skill that sends voice commands to your Raspberry Pi.

## Architecture

```
Alexa → Cloudflare Tunnel → Raspberry Pi → Home Assistant
         (free HTTPS)       (your app)      (your devices)
```

## Prerequisites

- [ ] Amazon Developer account (free): https://developer.amazon.com
- [ ] Cloudflare account (free): https://cloudflare.com
- [ ] Raspberry Pi with the smart-home app running

## Estimated time: 20-30 minutes

---

## Part 1: Expose Your Raspberry Pi (10 min)

### 1.1 Create Cloudflare Account and Tunnel

1. Go to https://one.dash.cloudflare.com and create a free account (select the Free plan)
2. Go to **Networks** → **Tunnels** (left sidebar)
4. Click **Create a tunnel**
5. Select **Cloudflared** and click **Next**
6. Name your tunnel: `smart-home`
7. Click **Save tunnel**
8. **Copy the tunnel token** - you'll need this for the RPi

### 1.2 Configure Public Hostname

1. Click on your tunnel → **Public Hostname** tab
2. Click **Add a public hostname**
3. Configure:
   - **Subdomain**: `smart-home` (or any name)
   - **Domain**: Select your domain, or use the free `cfargotunnel.com`
   - **Service Type**: `HTTP`
   - **URL**: `localhost:8080`
4. Click **Save hostname**

Your URL will be something like: `https://smart-home-abc123.cfargotunnel.com`

### 1.3 Install Tunnel on Raspberry Pi

```bash
# SSH into your RPi
ssh pi@raspberrypi.local

# Install cloudflared
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64 -o cloudflared
chmod +x cloudflared
sudo mv cloudflared /usr/local/bin/

# Install as service (replace YOUR_TUNNEL_TOKEN with the token from step 1.1)
sudo cloudflared service install YOUR_TUNNEL_TOKEN

# Enable and start
sudo systemctl enable cloudflared
sudo systemctl start cloudflared
```

### 1.4 Verify Tunnel Works

```bash
# From anywhere (your phone, another computer)
curl https://YOUR_TUNNEL_URL/health
# Should respond: {"status":"ok"}
```

---

## Part 2: Configure Auth Token (Security)

Since your endpoint is now public, add an authentication token:

### 2.1 Generate a secure token

```bash
openssl rand -hex 32
```

### 2.2 Add to your config.yaml

```yaml
audio:
  source: http
  http_addr: ":8080"
  auth_token: "your_generated_token_here"
```

### 2.3 Restart the app

```bash
docker compose restart
```

### 2.4 Test with token

```bash
curl -X POST "https://YOUR_TUNNEL_URL/text?token=your_token" \
  -d "turn on living room light"
```

---

## Part 3: Create Alexa Skill (15 min)

### 3.1 Create Skill

1. Go to https://developer.amazon.com/alexa/console/ask
2. Click **Create Skill**
3. Configure:
   - Skill name: `Home Control` (or any name)
   - Primary locale: `English (US)` or your preferred language
   - Type: **Custom**
   - Hosting: **Provision your own**
4. Click **Create Skill**
5. Template: **Start from Scratch**
6. Click **Continue with template**

### 3.2 Configure Invocation Name

1. In left menu: **Invocations** → **Skill Invocation Name**
2. Enter: `home` (or any word you prefer)
3. Click **Save**

**Note:** Avoid reserved words like: `alexa`, `amazon`, `echo`, `computer`, `trigger`, `launch`.

### 3.3 Import Interaction Model

1. In left menu: **Interaction Model** → **JSON Editor**
2. Delete everything and paste:

```json
{
  "interactionModel": {
    "languageModel": {
      "invocationName": "home",
      "intents": [
        {
          "name": "SmartHomeIntent",
          "slots": [
            {
              "name": "command",
              "type": "AMAZON.SearchQuery"
            }
          ],
          "samples": [
            "{command}",
            "to {command}",
            "please {command}"
          ]
        },
        {
          "name": "AMAZON.HelpIntent",
          "samples": []
        },
        {
          "name": "AMAZON.StopIntent",
          "samples": []
        },
        {
          "name": "AMAZON.CancelIntent",
          "samples": []
        }
      ]
    }
  }
}
```

4. Click **Save Model**
5. Click **Build Model** (wait 1-2 minutes)

### 3.4 Configure Endpoint

1. In left menu: **Endpoint**
2. Select **HTTPS**
3. Default Region: Enter your tunnel URL with token:
   ```
   https://YOUR_TUNNEL_URL/alexa?token=YOUR_AUTH_TOKEN
   ```
4. SSL Certificate: Select **My development endpoint is a sub-domain of a domain that has a wildcard certificate from a certificate authority**
5. Click **Save Endpoints**

### 3.5 Test in Console

1. Go to **Test** tab
2. Enable testing: **Development**
3. Type: `tell home to turn on living room light`
4. You should see a response

---

## Part 4: Test with Real Alexa

### With Echo device:
> "Alexa, tell home to turn on the living room light"

### With Alexa app:
1. Open Alexa app on your phone
2. Tap the Alexa button
3. Say: "tell home to turn off everything"

---

## Example Commands

- "Alexa, tell home to turn on the kitchen light"
- "Alexa, tell home to activate movie scene"
- "Alexa, tell home to turn off all lights"
- "Alexa, tell home to set bedroom light to 50 percent"

---

## Troubleshooting

### "Skill not found"
- Make sure you're using the same Amazon account in Developer Console and on your Echo/Alexa app

### "There was a problem with the requested skill's response"
- Check tunnel is running: `sudo systemctl status cloudflared`
- Check app logs: `docker compose logs -f`
- Verify the endpoint URL and token are correct

### Tunnel not connecting
```bash
# Check status
sudo systemctl status cloudflared

# View logs
sudo journalctl -u cloudflared -f

# Restart
sudo systemctl restart cloudflared
```

### Test endpoint manually
```bash
# With query parameter
curl -X POST "https://YOUR_TUNNEL_URL/alexa?token=YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "turn on living room light"

# With header
curl -X POST "https://YOUR_TUNNEL_URL/alexa" \
  -H "X-Auth-Token: YOUR_TOKEN" \
  -H "Content-Type: text/plain" \
  -d "turn on living room light"
```

---

## Useful Commands

```bash
# On Raspberry Pi

# Check tunnel status
sudo systemctl status cloudflared

# View tunnel logs
sudo journalctl -u cloudflared -f

# View app logs
docker compose logs -f

# Restart app
docker compose restart

# Restart tunnel
sudo systemctl restart cloudflared
```
