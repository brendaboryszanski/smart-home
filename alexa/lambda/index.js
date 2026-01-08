/**
 * AWS Lambda function for Alexa Skill
 *
 * Setup:
 * 1. Create Lambda function in AWS Console
 * 2. Runtime: Node.js 20.x
 * 3. Add trigger: Alexa Skills Kit
 * 4. Copy this code
 * 5. Set environment variables:
 *    - SMART_HOME_URL: Your Raspberry Pi endpoint URL
 *    - AUTH_TOKEN: Your authentication token from RPI setup
 */

const https = require('https');
const http = require('http');

const SMART_HOME_URL = process.env.SMART_HOME_URL || 'https://home.yourdomain.com';
const AUTH_TOKEN = process.env.AUTH_TOKEN;

exports.handler = async (event) => {
    console.log('Request:', JSON.stringify(event));

    const requestType = event.request.type;

    if (requestType === 'LaunchRequest') {
        return buildResponse('What would you like me to do?', false);
    }

    if (requestType === 'IntentRequest') {
        const intentName = event.request.intent.name;

        if (intentName === 'SmartHomeIntent') {
            const command = event.request.intent.slots?.command?.value;
            
            if (!command) {
                return buildResponse("I didn't understand the command", true);
            }

            try {
                await sendToSmartHome(command);
                return buildResponse(`Executing: ${command}`, true);
            } catch (error) {
                console.error('Error:', error);
                return buildResponse('There was an error executing the command', true);
            }
        }

        if (intentName === 'AMAZON.HelpIntent') {
            return buildResponse(
                'You can say things like: turn on the living room light, turn off everything, or activate movie scene',
                false
            );
        }

        if (intentName === 'AMAZON.StopIntent' || intentName === 'AMAZON.CancelIntent') {
            return buildResponse('Goodbye', true);
        }
    }

    return buildResponse("I didn't understand", true);
};

function sendToSmartHome(command) {
    return new Promise((resolve, reject) => {
        const url = new URL(SMART_HOME_URL + '/alexa');
        const client = url.protocol === 'https:' ? https : http;

        const headers = {
            'Content-Type': 'text/plain',
            'Content-Length': Buffer.byteLength(command)
        };

        // Add authentication token if configured
        if (AUTH_TOKEN) {
            headers['X-Auth-Token'] = AUTH_TOKEN;
        }

        const options = {
            hostname: url.hostname,
            port: url.port || (url.protocol === 'https:' ? 443 : 80),
            path: url.pathname,
            method: 'POST',
            headers: headers,
            timeout: 5000
        };

        const req = client.request(options, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => {
                console.log('Response:', data);
                resolve(data);
            });
        });

        req.on('error', reject);
        req.on('timeout', () => reject(new Error('timeout')));
        
        req.write(command);
        req.end();
    });
}

function buildResponse(text, shouldEnd) {
    return {
        version: '1.0',
        response: {
            outputSpeech: {
                type: 'PlainText',
                text: text
            },
            shouldEndSession: shouldEnd
        }
    };
}
