#!/bin/bash

echo "🐧 Takakrypt Ubuntu Setup"
echo "========================"

if [ "$EUID" -ne 0 ]; then
    echo "❌ Run as root: sudo ./ubuntu-setup.sh"
    exit 1
fi

echo "📦 Installing packages..."
apt update
apt install -y fuse3 libfuse3-dev golang-go mariadb-server sqlite3 nano vim

echo "👥 Creating users..."
useradd -m -s /bin/bash testuser1 || true
useradd -m -s /bin/bash testuser2 || true  
useradd -m -s /bin/bash dbadmin || true
echo "testuser1:password123" | chpasswd
echo "testuser2:password123" | chpasswd
echo "dbadmin:admin123" | chpasswd

echo "📁 Creating directories..."
mkdir -p /data/{sensitive,public,database}
mkdir -p /secure_storage/{sensitive,public,database}
chmod 755 /data /secure_storage

echo "🔧 Setting up FUSE..."
if ! getent group fuse >/dev/null; then
    groupadd fuse
fi
usermod -a -G fuse testuser1
usermod -a -G fuse testuser2
usermod -a -G fuse dbadmin
echo "user_allow_other" >> /etc/fuse.conf
modprobe fuse
chmod 666 /dev/fuse

echo "🗄️ Setting up MariaDB..."
systemctl enable mariadb
systemctl start mariadb
mysql -e "CREATE DATABASE IF NOT EXISTS testapp;"
mysql -e "CREATE USER IF NOT EXISTS 'appuser'@'localhost' IDENTIFIED BY 'apppass123';"
mysql -e "GRANT ALL PRIVILEGES ON testapp.* TO 'appuser'@'localhost';"
mysql testapp -e "
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50),
    email VARCHAR(100)
);
INSERT INTO users (username, email) VALUES 
('john_doe', 'john@example.com'),
('jane_smith', 'jane@example.com');
"

echo "📄 Creating test files..."
cat > /data/sensitive/confidential.txt << 'EOF'
CONFIDENTIAL DOCUMENT
====================
Employee data and trade secrets.
DO NOT SHARE WITHOUT AUTHORIZATION
EOF

cat > /data/public/readme.txt << 'EOF'
Public Information
==================
This is public information.
EOF

cat > /data/database/backup.sql << 'EOF'
-- Database backup
INSERT INTO users VALUES (999, 'secret_user', 'secret@company.com');
EOF

echo "✅ Ubuntu setup complete!"
echo ""
echo "Next: ./deploy-agent.sh"