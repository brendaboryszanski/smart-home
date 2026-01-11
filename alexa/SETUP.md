# Step-by-Step Guide: Alexa Skill + Lambda + Cloudflare Tunnel

> **Note:** This guide uses the recommended architecture: **Alexa → Lambda → RPI**.
> If you want to skip Lambda and connect Alexa directly to your RPI, see the "Direct Connection" section at the end.

## Prerequisites

- [ ] AWS account (free): https://aws.amazon.com/free
- [ ] Amazon Developer account (free): https://developer.amazon.com
- [ ] Cloudflare account (free): https://cloudflare.com
- [ ] Raspberry Pi with Docker installed

**Note:** You do NOT need to buy a domain. Cloudflare provides a free subdomain for your tunnel.

## Estimated time: 30-45 minutes

---

## Part 1: Cloudflare Tunnel Setup (10 min)

### 1.1 Create Cloudflare Account

1. Go to https://dash.cloudflare.com
2. Create a free account

### 1.2 Create a Tunnel (via Zero Trust Dashboard)

1. In Cloudflare dashboard, go to **Zero Trust** (left sidebar)
2. Go to **Networks** → **Tunnels**
3. Click **Create a tunnel**
4. Select **Cloudflared** and click **Next**
5. Name your tunnel: `smart-home`
6. Click **Save tunnel**
7. **Copy the tunnel token** - you'll need this for the RPi

### 1.3 Configure Public Hostname

After creating the tunnel:
1. Click on your tunnel → **Public Hostname** tab
2. Click **Add a public hostname**
3. Configure:
   - **Subdomain**: `smart-home` (or any name you want)
   - **Domain**: Select `cfargotunnel.com` (free) or your own domain if you have one
   - **Service Type**: `HTTP`
   - **URL**: `localhost:8080`
4. Click **Save hostname**

Your free URL will be: `https://smart-home-XXXXX.cfargotunnel.com`

---

## Part 2: Raspberry Pi + Tunnel (15 min)

### 2.1 Connect to your RPi
```bash
ssh pi@raspberrypi.local
```

### 2.2 Clone project (or copy files)
```bash
git clone https://github.com/your-username/smart-home.git
cd smart-home
```

### 2.3 Configure credentials
```bash
cp config.example.yaml config.yaml
nano config.yaml
# Add your API key for Gemini (or Anthropic) and Home Assistant URL/token
```

### 2.4 Start the service
```bash
docker compose up -d

# Verify it works
curl http://localhost:8080/health
# Should respond: {"status":"ok"}
```

### 2.5 Install and Run Cloudflare Tunnel

Use the tunnel token you copied from the Cloudflare dashboard:

```bash
# Install cloudflared
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64 -o cloudflared
chmod +x cloudflared
sudo mv cloudflared /usr/local/bin/

# Run with your tunnel token (replace YOUR_TUNNEL_TOKEN)
sudo cloudflared service install YOUR_TUNNEL_TOKEN

# Enable and start the service
sudo systemctl enable cloudflared
sudo systemctl start cloudflared
```

### 2.6 Verify tunnel
```bash
# From anywhere with internet (use your tunnel URL from Part 1)
curl https://smart-home-XXXXX.cfargotunnel.com/health
# Should respond: {"status":"ok"}
```

### 2.7 Configure Auth Token (Security)

Since your endpoint will be public, configure an authentication token:

**Generate a secure token:**
```bash
openssl rand -hex 32
```

**Add to environment:**
```bash
# Create .env file
echo "ALEXA_AUTH_TOKEN=your_generated_token_here" >> .env

# Restart with the token
docker compose down
docker compose up -d
```

**Save this token** - you'll need it when configuring the Alexa skill endpoint URL in Part 4.

---

## Part 3: AWS Lambda (10 min)

### 3.1 Create Lambda function

1. Go to https://console.aws.amazon.com/lambda
2. Region: **US East (N. Virginia)** `us-east-1` ← important for Alexa
3. Click "Create function"
4. Configure:
   - Function name: `smart-home-alexa`
   - Runtime: `Node.js 20.x`
   - Architecture: `x86_64`
5. Click "Create function"

### 3.2 Add code

1. In the function, scroll down to "Code source"
2. Open `index.js`
3. Delete everything and paste contents of `alexa/lambda/index.js`
4. Click "Deploy"

### 3.3 Configure environment variables

1. Tab "Configuration" → "Environment variables"
2. Click "Edit" → "Add environment variable"
3. Add both:
   - Key: `SMART_HOME_URL` → Value: `https://smart-home-XXXXX.cfargotunnel.com` (your tunnel URL)
   - Key: `AUTH_TOKEN` → Value: `your_token_from_step_2.7`
4. Save

### 3.4 Copy ARN

1. At the top, copy the function ARN
2. Looks like: `arn:aws:lambda:us-east-1:123456789:function:smart-home-alexa`
3. Save it, you'll need it for the skill

---

## Part 4: Alexa Skill (15 min)

### 4.1 Create Skill

1. Go to https://developer.amazon.com/alexa/console/ask
2. Click "Create Skill"
3. Configure:
   - Skill name: `Home Control`
   - Primary locale: `English (US)` or your preferred
   - Model: `Custom`
   - Hosting: `Provision your own`
4. Click "Create Skill"
5. Template: `Start from Scratch`
6. Click "Continue with template"

### 4.2 Configure Invocation Name

1. In left menu: "Invocations" → "Skill Invocation Name"
2. Enter: `amor` (or any word you prefer - see note below)
3. Save

**Note:** You can use any invocation name except reserved words like: `trigger`, `start`, `stop`, `launch`, `ask`, `tell`, `open`, `run`. Examples: `amor`, `casa`, `hogar`, `home`, `house`.

### 4.3 Import Interaction Model

1. In left menu: "Interaction Model" → "JSON Editor"
2. Delete everything
3. Paste contents of `alexa/interaction-model-en.json`
4. Click "Save Model"
5. Click "Build Model" (wait 1-2 min)

### 4.4 Connect to Lambda

1. In left menu: "Endpoint"
2. Select "AWS Lambda ARN"
3. Default Region: paste your Lambda ARN
4. Click "Save Endpoints"

### 4.5 Add trigger in Lambda

1. Go back to AWS Lambda Console
2. In your function, click "Add trigger"
3. Select "Alexa Skills Kit"
4. Skill ID: copy from Alexa Developer Console (in Endpoint section)
5. Click "Add"

### 4.6 Test in console

1. In Alexa Developer Console, go to "Test"
2. Enable testing: "Development"
3. Type: `tell amor to turn on living room light`
4. You should see the response

---

## Part 5: Test on real device

### With your Echo/Alexa:
> "Alexa, amor prende la luz del living"

### With Alexa app on phone:
1. Open Alexa app
2. Tap the Alexa button
3. Say: "amor apaga todo"

---

## Troubleshooting

### "Skill not found"
- Verify you're using the same Amazon account in Developer Console and on your Echo

### "Error executing command"
- Check tunnel is running: `sudo systemctl status cloudflared`
- Check logs: `docker compose logs -f`

### "Unknown command"
- Claude didn't understand the command
- Verify devices are synced

### Lambda timeout
- Increase timeout in Lambda: Configuration → General → Timeout → 10 seconds

---

## Useful commands

```bash
# On the Raspberry Pi

# Check tunnel status
sudo systemctl status cloudflared

# View tunnel logs
sudo journalctl -u cloudflared -f

# View assistant logs
docker compose logs -f

# Restart assistant
docker compose restart

# Test endpoint manually (with auth token)
curl -X POST https://YOUR_TUNNEL_URL/alexa?token=your_token_here -d "prende la luz"

# Or using header
curl -X POST https://YOUR_TUNNEL_URL/alexa \
  -H "X-Auth-Token: your_token_here" \
  -d "prende la luz"
```

---

## Done! 🎉

Now you can say:
- "Alexa, amor prende la luz del living"
- "Alexa, amor activa escena película"
- "Alexa, amor apaga todo"

---

## Alternative: Direct Connection (Without Lambda)

If you want to skip Lambda and connect Alexa directly to your RPI:

### Pros:
- Simpler setup (no AWS Lambda)
- One less component to maintain

### Cons:
- No Alexa signature verification
- Requires your token to be in the Alexa endpoint URL (visible in console)

### Setup:

1. Follow steps 1-2 from above (Cloudflare Tunnel + RPI with token)
2. Create Alexa Skill (Part 4 steps 4.1-4.3)
3. In step 4.4 (Endpoint), instead of Lambda:
   - Select: **HTTPS**
   - Default Region: `https://YOUR_TUNNEL_URL/alexa?token=your_token_here`
   - SSL Certificate: **My development endpoint is a sub-domain of a domain that has a wildcard certificate from a certificate authority**
4. Save and test

**Security note:** With this approach, your auth token is visible in the Alexa Developer Console. For home use this is acceptable, but Lambda provides better security.
