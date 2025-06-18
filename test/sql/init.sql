-- Инициализация тестовых данных для интеграционных тестов

-- Создание тестовых баз данных
CREATE DATABASE IF NOT EXISTS test_db_small;
CREATE DATABASE IF NOT EXISTS test_db_large;
CREATE DATABASE IF NOT EXISTS production_db;

-- Использование test_db_small
USE test_db_small;

-- Создание таблиц с тестовыми данными
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Вставка тестовых данных
INSERT INTO users (name, email) VALUES 
    ('John Doe', 'john@example.com'),
    ('Jane Smith', 'jane@example.com'),
    ('Bob Wilson', 'bob@example.com');

INSERT INTO posts (user_id, title, content) VALUES 
    (1, 'First Post', 'This is the content of the first post'),
    (1, 'Second Post', 'This is the content of the second post'),
    (2, 'Jane\'s Post', 'Jane\'s first blog post'),
    (3, 'Bob\'s Article', 'An interesting article by Bob');

-- Использование test_db_large для тестирования производительности
USE test_db_large;

-- Создание таблиц с большим количеством данных
CREATE TABLE large_table (
    id INT AUTO_INCREMENT PRIMARY KEY,
    data VARCHAR(255) NOT NULL,
    random_number INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Вставка большого количества тестовых данных
INSERT INTO large_table (data, random_number) 
SELECT 
    CONCAT('test_data_', n),
    FLOOR(RAND() * 10000)
FROM (
    SELECT a.N + b.N * 10 + c.N * 100 + 1 n
    FROM 
        (SELECT 0 AS N UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) a
        CROSS JOIN (SELECT 0 AS N UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) b
        CROSS JOIN (SELECT 0 AS N UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) c
) numbers
WHERE n <= 1000;

-- Использование production_db для тестирования безопасности
USE production_db;

-- Имитация "критической" БД
CREATE TABLE critical_data (
    id INT AUTO_INCREMENT PRIMARY KEY,
    sensitive_info VARCHAR(255) NOT NULL,
    classification ENUM('PUBLIC', 'INTERNAL', 'CONFIDENTIAL', 'SECRET') DEFAULT 'CONFIDENTIAL',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO critical_data (sensitive_info, classification) VALUES 
    ('Production user data', 'CONFIDENTIAL'),
    ('Financial records', 'SECRET'),
    ('Customer information', 'CONFIDENTIAL'),
    ('Internal procedures', 'INTERNAL');

-- Создание представлений для тестирования
CREATE VIEW user_post_count AS
SELECT 
    u.name,
    u.email,
    COUNT(p.id) as post_count
FROM test_db_small.users u
LEFT JOIN test_db_small.posts p ON u.id = p.user_id
GROUP BY u.id, u.name, u.email;

-- Создание индексов для тестирования
CREATE INDEX idx_large_table_data ON large_table(data);
CREATE INDEX idx_large_table_number ON large_table(random_number);

-- Создание хранимой процедуры для тестирования
DELIMITER $$
CREATE PROCEDURE test_db_small.GetUserPosts(IN user_id INT)
BEGIN
    SELECT p.title, p.content, p.created_at
    FROM test_db_small.posts p
    WHERE p.user_id = user_id
    ORDER BY p.created_at DESC;
END $$
DELIMITER ;

-- Создание функции для тестирования
DELIMITER $$
CREATE FUNCTION test_db_small.GetUserPostCount(user_id INT) RETURNS INT
READS SQL DATA
DETERMINISTIC
BEGIN
    DECLARE post_count INT DEFAULT 0;
    SELECT COUNT(*) INTO post_count
    FROM test_db_small.posts p
    WHERE p.user_id = user_id;
    RETURN post_count;
END $$
DELIMITER ;

-- Создание триггера для тестирования
DELIMITER $$
CREATE TRIGGER test_db_small.update_user_timestamp
    BEFORE UPDATE ON test_db_small.users
    FOR EACH ROW
BEGIN
    SET NEW.created_at = CURRENT_TIMESTAMP;
END $$
DELIMITER ;
