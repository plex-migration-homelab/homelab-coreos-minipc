#!/bin/bash

# Create setup directory for templates
mkdir -p /home/core/setup
cp -r /usr/share/compose-setup /home/core/setup
chown -R core:core /home/core/setup

# Create the production compose directory structure
mkdir -p /srv/containers/media
mkdir -p /srv/containers/web
mkdir -p /srv/containers/cloud

# Copy compose files to production location
cp -r /usr/share/compose-setup/*.yml /srv/containers/
cp /usr/share/compose-setup/.env.example /srv/containers/.env.example

# Create appdata directory structure
mkdir -p /var/lib/containers/appdata/plex
mkdir -p /var/lib/containers/appdata/jellyfin
mkdir -p /var/lib/containers/appdata/tautulli
mkdir -p /var/lib/containers/appdata/overseerr
mkdir -p /var/lib/containers/appdata/wizarr
mkdir -p /var/lib/containers/appdata/organizr
mkdir -p /var/lib/containers/appdata/homepage
mkdir -p /var/lib/containers/appdata/nextcloud
mkdir -p /var/lib/containers/appdata/immich
mkdir -p /var/lib/containers/appdata/postgres
mkdir -p /var/lib/containers/appdata/redis

# Set appropriate ownership (dockeruser:dockeruser)
# Note: dockeruser may not exist yet at first boot, so we defer this to post-install
# For now, set to core:core or root:root for initial setup
chown -R core:core /srv/containers
chown -R core:core /var/lib/containers/appdata
