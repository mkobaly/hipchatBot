# Simple Hipchat bot to local Teamcity server

## Summary
This P.O.C is a work in progress but its currently functional. The hipchat bot will allow your team members to kick of builds against a local Teamcity server that they might not have web access to. Useful for remote teams that work in different locations.

## Details

### Step 1

We are using ngrok to support web kooks. Download ngrok and start it up using below command. This will forward all http requests to port 8030 (port the hipchat bot will be running on)

    ngrok http 8030

This will generate a new public facing HTTPS URL that we will use later to configure our Hipchat integration.

### Step 2

- Log into hipchat and select the room you want to configure the integration for
- Create new integration and configure it according to the screenshot below

![](hipchatintegration.png)

### Step 3

In order for the hipchatbot to run it needs a configuration file. Copy config.yaml.template and rename it to config.yaml. It must be placed in same folder as hipchatbot application

- You will need the hipchart POST api url gotten from the hipchat integration page
- You will need the ngrok URL that is displayed in Step 1 when you ran ngrok
- You will need your Teamcity URL and credentials

start up the hipchat bot


    hipchatBot




