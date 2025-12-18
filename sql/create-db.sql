CREATE DATABASE IF NOT EXISTS mhs_main;
GRANT ALL PRIVILEGES ON mhs_main.* TO 'mhserver'@'localhost';

CREATE DATABASE IF NOT EXISTS mhs_main_test;
GRANT ALL PRIVILEGES ON mhs_main_test.* TO 'mhserver_tests'@'localhost';