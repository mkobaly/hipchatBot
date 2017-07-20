# Simple Hipchat bot to local Teamcity server

## Summary
This POC is a work in progress but its currently functional. The hipchat bot will allow your team
to kick of builds against a local Teamcity server that they might not have web access to

## Details
We are using ngrok to support web kooks. 

ngrok http 8030

This will generate a new public facing URL that you will paste into the hipchat bot configuration

![](hipchatintegratoin.png)
