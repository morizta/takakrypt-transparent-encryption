#!/usr/bin/env node
/**
 * Node.js Application Access Example - Transparent Encryption Test
 * This simulates a Node.js web API accessing MariaDB with transparent encryption
 */

const mysql = require('mysql2/promise');
const express = require('express');
const app = express();

app.use(express.json());

class ECommerceAPI {
    constructor() {
        this.pool = null;
    }

    async initialize() {
        try {
            // Create connection pool
            this.pool = mysql.createPool({
                host: 'localhost',
                user: 'root',
                password: '', // Adjust if you have a password
                database: 'node_app_db',
                waitForConnections: true,
                connectionLimit: 10,
                queueLimit: 0
            });

            console.log('✓ Connected to MariaDB pool');
            await this.setupDatabase();
        } catch (error) {
            console.error('✗ Database connection failed:', error.message);
            throw error;
        }
    }

    async setupDatabase() {
        try {
            // Create database if not exists
            await this.pool.execute('CREATE DATABASE IF NOT EXISTS node_app_db');
            await this.pool.execute('USE node_app_db');

            // Create products table
            await this.pool.execute(`
                CREATE TABLE IF NOT EXISTS products (
                    id INT AUTO_INCREMENT PRIMARY KEY,
                    name VARCHAR(255) NOT NULL,
                    description TEXT,
                    price DECIMAL(10,2) NOT NULL,
                    inventory_count INT DEFAULT 0,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            `);

            // Create user sessions table (contains sensitive data)
            await this.pool.execute(`
                CREATE TABLE IF NOT EXISTS user_sessions (
                    id INT AUTO_INCREMENT PRIMARY KEY,
                    session_id VARCHAR(255) UNIQUE NOT NULL,
                    user_data JSON,
                    personal_info JSON,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    expires_at TIMESTAMP
                )
            `);

            // Create transactions table
            await this.pool.execute(`
                CREATE TABLE IF NOT EXISTS transactions (
                    id INT AUTO_INCREMENT PRIMARY KEY,
                    session_id VARCHAR(255),
                    transaction_data JSON,
                    payment_details JSON,
                    amount DECIMAL(10,2),
                    status VARCHAR(50),
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    FOREIGN KEY (session_id) REFERENCES user_sessions(session_id)
                )
            `);

            console.log('✓ Database schema created');
        } catch (error) {
            console.error('✗ Database setup failed:', error.message);
            throw error;
        }
    }

    // API Routes

    async createUserSession(userData, personalInfo) {
        try {
            const sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
            const expiresAt = new Date(Date.now() + 24 * 60 * 60 * 1000); // 24 hours

            const [result] = await this.pool.execute(
                'INSERT INTO user_sessions (session_id, user_data, personal_info, expires_at) VALUES (?, ?, ?, ?)',
                [sessionId, JSON.stringify(userData), JSON.stringify(personalInfo), expiresAt]
            );

            console.log(`✓ Created user session: ${sessionId}`);
            return { sessionId, id: result.insertId };
        } catch (error) {
            console.error('✗ Failed to create session:', error.message);
            throw error;
        }
    }

    async addProduct(name, description, price, inventory) {
        try {
            const [result] = await this.pool.execute(
                'INSERT INTO products (name, description, price, inventory_count) VALUES (?, ?, ?, ?)',
                [name, description, price, inventory]
            );

            console.log(`✓ Added product: ${name} (ID: ${result.insertId})`);
            return { id: result.insertId, name, price };
        } catch (error) {
            console.error('✗ Failed to add product:', error.message);
            throw error;
        }
    }

    async processTransaction(sessionId, items, paymentDetails) {
        const connection = await this.pool.getConnection();
        
        try {
            await connection.beginTransaction();

            // Calculate total
            let total = 0;
            for (const item of items) {
                const [products] = await connection.execute(
                    'SELECT price, inventory_count FROM products WHERE id = ?',
                    [item.productId]
                );
                
                if (products.length === 0) {
                    throw new Error(`Product ${item.productId} not found`);
                }

                const product = products[0];
                if (product.inventory_count < item.quantity) {
                    throw new Error(`Insufficient inventory for product ${item.productId}`);
                }

                total += product.price * item.quantity;

                // Update inventory
                await connection.execute(
                    'UPDATE products SET inventory_count = inventory_count - ? WHERE id = ?',
                    [item.quantity, item.productId]
                );
            }

            // Create transaction record
            const transactionData = {
                items: items,
                total: total,
                timestamp: new Date().toISOString()
            };

            const [result] = await connection.execute(
                'INSERT INTO transactions (session_id, transaction_data, payment_details, amount, status) VALUES (?, ?, ?, ?, ?)',
                [sessionId, JSON.stringify(transactionData), JSON.stringify(paymentDetails), total, 'completed']
            );

            await connection.commit();
            console.log(`✓ Processed transaction: $${total} (ID: ${result.insertId})`);
            
            return { 
                transactionId: result.insertId, 
                total: total, 
                status: 'completed' 
            };

        } catch (error) {
            await connection.rollback();
            console.error('✗ Transaction failed:', error.message);
            throw error;
        } finally {
            connection.release();
        }
    }

    async getUserAnalytics(sessionId) {
        try {
            const [sessions] = await this.pool.execute(
                'SELECT user_data, personal_info FROM user_sessions WHERE session_id = ?',
                [sessionId]
            );

            const [transactions] = await this.pool.execute(
                'SELECT transaction_data, amount, created_at FROM transactions WHERE session_id = ?',
                [sessionId]
            );

            if (sessions.length === 0) {
                throw new Error('Session not found');
            }

            const analytics = {
                session: {
                    userData: JSON.parse(sessions[0].user_data),
                    personalInfo: JSON.parse(sessions[0].personal_info)
                },
                transactions: transactions.map(t => ({
                    data: JSON.parse(t.transaction_data),
                    amount: parseFloat(t.amount),
                    date: t.created_at
                })),
                totalSpent: transactions.reduce((sum, t) => sum + parseFloat(t.amount), 0),
                transactionCount: transactions.length
            };

            console.log(`✓ Retrieved analytics for session: ${sessionId}`);
            return analytics;

        } catch (error) {
            console.error('✗ Failed to get analytics:', error.message);
            throw error;
        }
    }

    async getProducts() {
        try {
            const [products] = await this.pool.execute(
                'SELECT id, name, description, price, inventory_count FROM products ORDER BY name'
            );

            console.log(`✓ Retrieved ${products.length} products`);
            return products;
        } catch (error) {
            console.error('✗ Failed to get products:', error.message);
            throw error;
        }
    }

    async close() {
        if (this.pool) {
            await this.pool.end();
            console.log('✓ Database pool closed');
        }
    }
}

// Initialize API
const api = new ECommerceAPI();

// Express routes
app.post('/api/session', async (req, res) => {
    try {
        const { userData, personalInfo } = req.body;
        const session = await api.createUserSession(userData, personalInfo);
        res.json({ success: true, session });
    } catch (error) {
        res.status(500).json({ success: false, error: error.message });
    }
});

app.post('/api/products', async (req, res) => {
    try {
        const { name, description, price, inventory } = req.body;
        const product = await api.addProduct(name, description, price, inventory);
        res.json({ success: true, product });
    } catch (error) {
        res.status(500).json({ success: false, error: error.message });
    }
});

app.get('/api/products', async (req, res) => {
    try {
        const products = await api.getProducts();
        res.json({ success: true, products });
    } catch (error) {
        res.status(500).json({ success: false, error: error.message });
    }
});

app.post('/api/transaction', async (req, res) => {
    try {
        const { sessionId, items, paymentDetails } = req.body;
        const transaction = await api.processTransaction(sessionId, items, paymentDetails);
        res.json({ success: true, transaction });
    } catch (error) {
        res.status(500).json({ success: false, error: error.message });
    }
});

app.get('/api/analytics/:sessionId', async (req, res) => {
    try {
        const { sessionId } = req.params;
        const analytics = await api.getUserAnalytics(sessionId);
        res.json({ success: true, analytics });
    } catch (error) {
        res.status(500).json({ success: false, error: error.message });
    }
});

// Test function
async function runApplicationTest() {
    console.log('=== Node.js Application Access Example ===');
    console.log('Simulating Node.js API accessing MariaDB with transparent encryption');
    console.log('');

    try {
        await api.initialize();

        console.log('\n--- Product Management Simulation ---');
        // Add test products
        const product1 = await api.addProduct(
            'Premium Laptop', 
            'High-performance laptop with encryption', 
            1299.99, 
            50
        );

        const product2 = await api.addProduct(
            'Security Software', 
            'Enterprise security suite', 
            299.99, 
            100
        );

        const product3 = await api.addProduct(
            'VPN Service', 
            'Annual VPN subscription', 
            89.99, 
            1000
        );

        console.log('\n--- User Session Management Simulation ---');
        // Create user sessions with sensitive data
        const session1 = await api.createUserSession(
            {
                userId: 'user_12345',
                username: 'john_secure',
                preferences: { theme: 'dark', notifications: true }
            },
            {
                fullName: 'John Security Expert',
                email: 'john@securecompany.com',
                ssn: '123-45-6789',
                creditCard: '4532-1234-5678-9012',
                address: '123 Encryption Lane, Secure City, SC 12345'
            }
        );

        const session2 = await api.createUserSession(
            {
                userId: 'user_67890',
                username: 'jane_admin',
                preferences: { theme: 'light', notifications: false }
            },
            {
                fullName: 'Jane Administrator',
                email: 'jane@techcorp.com',
                ssn: '987-65-4321',
                creditCard: '5555-4444-3333-2222',
                address: '456 Database Drive, Tech City, TC 67890'
            }
        );

        console.log('\n--- Transaction Processing Simulation ---');
        // Process transactions
        const transaction1 = await api.processTransaction(
            session1.sessionId,
            [
                { productId: product1.id, quantity: 1 },
                { productId: product2.id, quantity: 1 }
            ],
            {
                cardNumber: '**** **** **** 9012',
                expiryDate: '12/25',
                cvv: '***',
                billingAddress: '123 Encryption Lane'
            }
        );

        const transaction2 = await api.processTransaction(
            session2.sessionId,
            [
                { productId: product2.id, quantity: 2 },
                { productId: product3.id, quantity: 1 }
            ],
            {
                cardNumber: '**** **** **** 2222',
                expiryDate: '06/26',
                cvv: '***',
                billingAddress: '456 Database Drive'
            }
        );

        console.log('\n--- Data Analytics Simulation ---');
        // Get user analytics
        const analytics1 = await api.getUserAnalytics(session1.sessionId);
        console.log(`User ${analytics1.session.personalInfo.fullName}:`);
        console.log(`- Transactions: ${analytics1.transactionCount}`);
        console.log(`- Total Spent: $${analytics1.totalSpent}`);

        const analytics2 = await api.getUserAnalytics(session2.sessionId);
        console.log(`User ${analytics2.session.personalInfo.fullName}:`);
        console.log(`- Transactions: ${analytics2.transactionCount}`);
        console.log(`- Total Spent: $${analytics2.totalSpent}`);

        console.log('\n--- Bulk Operations Simulation ---');
        // Simulate multiple concurrent operations
        const bulkPromises = [];
        for (let i = 0; i < 5; i++) {
            const promise = api.createUserSession(
                { userId: `bulk_user_${i}`, batch: true },
                {
                    fullName: `Bulk User ${i}`,
                    email: `bulk${i}@test.com`,
                    ssn: `555-44-${3000 + i}`,
                    creditCard: `4532-${1000 + i}-5678-9012`
                }
            );
            bulkPromises.push(promise);
        }

        const bulkResults = await Promise.all(bulkPromises);
        console.log(`✓ Created ${bulkResults.length} bulk user sessions`);

        console.log('\n--- API Test Complete ---');
        console.log('All Node.js application operations completed successfully!');
        console.log('Sensitive data is transparently encrypted in the filesystem.');

    } catch (error) {
        console.error('✗ Application error:', error.message);
    }
}

// CLI execution
if (require.main === module) {
    runApplicationTest()
        .then(() => {
            console.log('\nTest completed. Use Ctrl+C to exit.');
        })
        .catch(error => {
            console.error('Test failed:', error.message);
            process.exit(1);
        });
} else {
    // Start as web server
    const PORT = 3000;
    api.initialize().then(() => {
        app.listen(PORT, () => {
            console.log(`Node.js API server running on port ${PORT}`);
            console.log('Test endpoints:');
            console.log('- POST /api/session - Create user session');
            console.log('- POST /api/products - Add product');
            console.log('- GET /api/products - List products');
            console.log('- POST /api/transaction - Process transaction');
            console.log('- GET /api/analytics/:sessionId - Get user analytics');
        });
    }).catch(error => {
        console.error('Failed to start server:', error.message);
        process.exit(1);
    });
}

module.exports = { ECommerceAPI, app };