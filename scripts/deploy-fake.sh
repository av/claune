#!/bin/bash

echo "INITIALIZING WS_FTP PRO 95..."
sleep 1
echo "CONNECTING TO ftp.geocities.com ON PORT 21..."
sleep 1.5
echo "USER: xX_Everlier_Xx"
sleep 0.5
echo "PASS: *********"
sleep 1
echo "230 User xX_Everlier_Xx logged in."
sleep 0.5
echo "SYST"
sleep 0.5
echo "215 UNIX Type: L8"
sleep 0.5
echo "TYPE I"
sleep 0.5
echo "200 Type set to I."
sleep 0.5
echo "PASV"
sleep 0.5
echo "227 Entering Passive Mode (192,168,1,100,14,178)."
sleep 0.5

echo ""
echo ">>> UPLOADING TO GEOCITIES: /Area51/Vault/1337/ <<<"
echo ""

files=(
  "index.html"
  "splash.html"
  "main.html"
  "nav.html"
  "links.html"
  "guestbook.html"
  "webmaster.html"
  "faq.html"
  "awards.html"
  "features.html"
  "install.html"
  "commands.html"
  "sitemap.html"
  "downloads.html"
  "404.html"
  "assets/images/stars.gif"
  "assets/sounds/bg.mid"
  "assets/downloads/claune-setup.exe"
)

for file in "${files[@]}"; do
  echo "STOR $file"
  sleep $((RANDOM % 2 + 1))
  echo "150 Opening BINARY mode data connection for $file."
  sleep 0.5
  echo "226 Transfer complete. (SUCCESS!)"
done

echo ""
echo "ALL FILES UPLOADED SUCCESSFULLY."
echo "DON'T FORGET TO UPDATE YOUR WEBRING HTML!!!"
echo "DISCONNECTING..."
sleep 1
echo "Goodbye."
