#!/bin/bash
# Copy dotfiles directory to core user's home
cp -a /usr/share/bluebuild/dotfiles/. /home/core/
echo "Dotfiles copied to /home/core/"
chown -R core:core /home/core/
echo "Dotfiles ownership changed to core user."