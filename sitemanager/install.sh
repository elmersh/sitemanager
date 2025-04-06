#!/bin/bash
if [ -f /usr/local/bin/sm ]; then
    rm /usr/local/bin/sm
fi
cp sm /usr/local/bin/
chmod +x /usr/local/bin/sm
echo "SiteManager instalado en /usr/local/bin/sm"
