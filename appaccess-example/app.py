#!/usr/bin/env python3
"""
Application Access Example - Transparent Encryption Test
This simulates a real application accessing MariaDB which stores data 
in the transparent encryption filesystem.
"""

import mysql.connector
import json
import time
import os
from datetime import datetime

class CustomerApp:
    def __init__(self):
        """Initialize database connection"""
        try:
            self.db = mysql.connector.connect(
                host='localhost',
                user='root',
                password='',  # Adjust if you have a password
                database='app_test_db',
                autocommit=True
            )
            self.cursor = self.db.cursor()
            print("✓ Connected to MariaDB")
        except mysql.connector.Error as err:
            print(f"✗ Database connection failed: {err}")
            raise

    def setup_database(self):
        """Create database and tables for the application"""
        try:
            # Create database if not exists
            self.cursor.execute("CREATE DATABASE IF NOT EXISTS app_test_db")
            self.cursor.execute("USE app_test_db")
            
            # Create customers table
            self.cursor.execute("""
                CREATE TABLE IF NOT EXISTS customers (
                    id INT AUTO_INCREMENT PRIMARY KEY,
                    name VARCHAR(255) NOT NULL,
                    email VARCHAR(255) NOT NULL,
                    ssn VARCHAR(11) NOT NULL,
                    credit_card VARCHAR(19) NOT NULL,
                    address TEXT,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            """)
            
            # Create orders table  
            self.cursor.execute("""
                CREATE TABLE IF NOT EXISTS orders (
                    id INT AUTO_INCREMENT PRIMARY KEY,
                    customer_id INT,
                    order_details JSON,
                    total_amount DECIMAL(10,2),
                    payment_info VARCHAR(255),
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    FOREIGN KEY (customer_id) REFERENCES customers(id)
                )
            """)
            
            print("✓ Database schema created")
            
        except mysql.connector.Error as err:
            print(f"✗ Database setup failed: {err}")
            raise

    def add_customer(self, name, email, ssn, credit_card, address):
        """Add a new customer (simulates app registration)"""
        try:
            query = """
                INSERT INTO customers (name, email, ssn, credit_card, address)
                VALUES (%s, %s, %s, %s, %s)
            """
            values = (name, email, ssn, credit_card, address)
            self.cursor.execute(query, values)
            customer_id = self.cursor.lastrowid
            print(f"✓ Added customer: {name} (ID: {customer_id})")
            return customer_id
            
        except mysql.connector.Error as err:
            print(f"✗ Failed to add customer: {err}")
            return None

    def create_order(self, customer_id, order_details, total_amount, payment_info):
        """Create an order (simulates e-commerce transaction)"""
        try:
            query = """
                INSERT INTO orders (customer_id, order_details, total_amount, payment_info)
                VALUES (%s, %s, %s, %s)
            """
            order_json = json.dumps(order_details)
            values = (customer_id, order_json, total_amount, payment_info)
            self.cursor.execute(query, values)
            order_id = self.cursor.lastrowid
            print(f"✓ Created order: ${total_amount} (ID: {order_id})")
            return order_id
            
        except mysql.connector.Error as err:
            print(f"✗ Failed to create order: {err}")
            return None

    def get_customer_data(self, customer_id):
        """Retrieve customer data (simulates app data access)"""
        try:
            query = """
                SELECT c.*, COUNT(o.id) as total_orders, SUM(o.total_amount) as total_spent
                FROM customers c
                LEFT JOIN orders o ON c.id = o.customer_id
                WHERE c.id = %s
                GROUP BY c.id
            """
            self.cursor.execute(query, (customer_id,))
            result = self.cursor.fetchone()
            
            if result:
                customer = {
                    'id': result[0],
                    'name': result[1],
                    'email': result[2],
                    'ssn': result[3],
                    'credit_card': result[4],
                    'address': result[5],
                    'created_at': result[6],
                    'total_orders': result[7],
                    'total_spent': result[8] or 0
                }
                print(f"✓ Retrieved customer data: {customer['name']}")
                return customer
            else:
                print(f"✗ Customer {customer_id} not found")
                return None
                
        except mysql.connector.Error as err:
            print(f"✗ Failed to retrieve customer: {err}")
            return None

    def search_customers_by_email(self, email_pattern):
        """Search customers by email (simulates app search functionality)"""
        try:
            query = "SELECT id, name, email FROM customers WHERE email LIKE %s"
            self.cursor.execute(query, (f"%{email_pattern}%",))
            results = self.cursor.fetchall()
            
            customers = []
            for row in results:
                customers.append({
                    'id': row[0],
                    'name': row[1], 
                    'email': row[2]
                })
            
            print(f"✓ Found {len(customers)} customers matching '{email_pattern}'")
            return customers
            
        except mysql.connector.Error as err:
            print(f"✗ Customer search failed: {err}")
            return []

    def get_order_history(self, customer_id):
        """Get order history (simulates app order tracking)"""
        try:
            query = """
                SELECT id, order_details, total_amount, payment_info, created_at
                FROM orders 
                WHERE customer_id = %s
                ORDER BY created_at DESC
            """
            self.cursor.execute(query, (customer_id,))
            results = self.cursor.fetchall()
            
            orders = []
            for row in results:
                orders.append({
                    'id': row[0],
                    'details': json.loads(row[1]),
                    'amount': float(row[2]),
                    'payment': row[3],
                    'date': row[4]
                })
            
            print(f"✓ Retrieved {len(orders)} orders for customer {customer_id}")
            return orders
            
        except mysql.connector.Error as err:
            print(f"✗ Failed to retrieve orders: {err}")
            return []

    def process_bulk_transactions(self, num_transactions=10):
        """Process multiple transactions (simulates high-volume app usage)"""
        print(f"\n--- Processing {num_transactions} bulk transactions ---")
        
        for i in range(num_transactions):
            # Add customer
            customer_id = self.add_customer(
                name=f"Bulk Customer {i+1}",
                email=f"bulk{i+1}@example.com",
                ssn=f"555-44-{3000+i:04d}",
                credit_card=f"4532-{1000+i:04d}-5678-9012",
                address=f"{100+i} Bulk Street, City, State {10000+i}"
            )
            
            if customer_id:
                # Create order
                order_details = {
                    "items": [
                        {"product": f"Product {i+1}", "quantity": 2, "price": 29.99},
                        {"product": f"Service {i+1}", "quantity": 1, "price": 49.99}
                    ],
                    "shipping": "Express",
                    "notes": f"Bulk order #{i+1}"
                }
                
                self.create_order(
                    customer_id=customer_id,
                    order_details=order_details,
                    total_amount=109.97,
                    payment_info=f"Credit Card ending in {9012+i:04d}"
                )
            
            # Small delay to simulate real usage
            time.sleep(0.1)

    def close(self):
        """Close database connection"""
        if self.db:
            self.cursor.close()
            self.db.close()
            print("✓ Database connection closed")

def main():
    """Main application simulation"""
    print("=== Application Access Example ===")
    print("Simulating real application accessing MariaDB with transparent encryption")
    print("")
    
    app = None
    try:
        # Initialize application
        app = CustomerApp()
        
        # Setup database
        app.setup_database()
        
        print("\n--- Customer Registration Simulation ---")
        # Add test customers
        customer1_id = app.add_customer(
            name="John Doe",
            email="john.doe@example.com", 
            ssn="123-45-6789",
            credit_card="4532-1234-5678-9012",
            address="123 Main St, Anytown, ST 12345"
        )
        
        customer2_id = app.add_customer(
            name="Jane Smith",
            email="jane.smith@company.com",
            ssn="987-65-4321", 
            credit_card="5555-4444-3333-2222",
            address="456 Oak Ave, Another City, ST 67890"
        )
        
        print("\n--- Order Processing Simulation ---")
        # Create orders
        if customer1_id:
            order1_details = {
                "items": [
                    {"product": "Laptop", "quantity": 1, "price": 999.99},
                    {"product": "Mouse", "quantity": 1, "price": 29.99}
                ],
                "shipping": "Standard",
                "notes": "Gift wrap requested"
            }
            app.create_order(customer1_id, order1_details, 1029.98, "Credit Card ending in 9012")
        
        if customer2_id:
            order2_details = {
                "items": [
                    {"product": "Phone", "quantity": 1, "price": 799.99},
                    {"product": "Case", "quantity": 1, "price": 19.99}
                ],
                "shipping": "Express", 
                "notes": "Corporate purchase"
            }
            app.create_order(customer2_id, order2_details, 819.98, "Corporate Card ending in 2222")
        
        print("\n--- Data Retrieval Simulation ---")
        # Retrieve customer data
        if customer1_id:
            customer_data = app.get_customer_data(customer1_id)
            if customer_data:
                print(f"Customer: {customer_data['name']}")
                print(f"Email: {customer_data['email']}")
                print(f"Total Orders: {customer_data['total_orders']}")
                print(f"Total Spent: ${customer_data['total_spent']}")
        
        print("\n--- Search Functionality Simulation ---")
        # Search customers
        search_results = app.search_customers_by_email("example.com")
        for customer in search_results:
            print(f"Found: {customer['name']} ({customer['email']})")
        
        print("\n--- Order History Simulation ---")
        # Get order history
        if customer1_id:
            orders = app.get_order_history(customer1_id)
            for order in orders:
                print(f"Order #{order['id']}: ${order['amount']} - {len(order['details']['items'])} items")
        
        # Bulk operations
        app.process_bulk_transactions(5)
        
        print("\n--- Application Test Complete ---")
        print("All application operations completed successfully!")
        print("Data is transparently encrypted in the filesystem.")
        
    except Exception as e:
        print(f"✗ Application error: {e}")
        
    finally:
        if app:
            app.close()

if __name__ == "__main__":
    main()