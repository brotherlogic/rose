sudo apt update

git config --global user.email 'brotherlogic-automation@gmail.com'
git config --global user.name 'Brotherlogic Automation'
tic -x ghostty.terminfo

# Install tmux and emacs
sudo apt-get install -y emacs tmux

sudo sh -c "$(curl -fsLS get.chezmoi.io)" -- -b /usr/local/bin


# Install antigravity
curl -fsSL https://antigravity.google/cli/install.sh -o install_antigravity.sh
bash install_antigravity.sh
rm install_antigravity.sh

# Set git identity
git config --global user.email "brotherlogicautomation@gmail.com"
git config --global user.name "Brotherlogic Automation"

TMUX_BLOCK=$(cat << 'EOF'
if [ -z "$TMUX" ] && [ -n "$PS1" ]; then
  cd /workspaces/rose
  /workspaces/rose/start-tmux.sh && tmux attach-session -t rose
fi
EOF
)

grep -q "tmux attach-session" ~/.zshrc || echo "$TMUX_BLOCK" >> ~/.zshrc
grep -q "tmux attach-session" ~/.bashrc || echo "$TMUX_BLOCK" >> ~/.bashrc

# Ensure the session is created
/workspaces/rose/start-tmux.sh
