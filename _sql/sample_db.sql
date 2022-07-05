-- Adminer 4.2.1 MySQL dump

SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

DROP TABLE IF EXISTS `items`;
CREATE TABLE `items` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `title` VARCHAR(255) NOT NULL,
  `description` TEXT NOT NULL,
  `updated` VARCHAR(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO
    `items` (`id`, `title`, `description`, `updated`)
VALUES
    (1,	'database/sql',	'Рассказать про базы данных',	'rvasily'),
    (2,	'memcache',	'Рассказать про мемкеш с примером использования', NULL);

DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `user_id` INT(11) NOT NULL AUTO_INCREMENT,
  `login` VARCHAR(255) NOT NULL,
  `password` VARCHAR(255) NOT NULL,
  `email` VARCHAR(255) NOT NULL,
  `info` TEXT NOT NULL,
  `updated` VARCHAR(255) DEFAULT NULL,
  PRIMARY KEY (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO
    `users`(`user_id`, `login`, `password`, `email`, `info`, `updated`)
VALUES
    (1,	'rvasily', 'love', 'rvasily@example.com', 'none', NULL);
