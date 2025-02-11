# CloudStorage

CloudStorage is a cloud-based file storage system that allows users to upload, organize, and manage their files. Built on a microservices architecture, it provides a secure and scalable solution for file storage and management.

## Features

- **File Upload and Management**: Users can upload, organize, and manage their files efficiently.
- **Secure Access**: Robust authentication and authorization mechanisms ensure the security of user data.
- **Scalable Storage**: Designed to handle large volumes of files with ease.
- **User-Friendly Interface**: Intuitive web interface for seamless interaction with the system.

## Project Structure

The repository is organized into the following main directories and files:

- **ApiGateway**: Contains the API gateway configuration and code.
- **Services**: Includes various microservices that power the application.
  - **Api Gateway**: Routes requests between clients and internal services for efficient interaction.
  - **Authentication Service**: Manages user access and ensures the security of their data.
  - **File Management Service**: Handles requests related to user file management.
  - **File Storage Service**: Allows users to store their files on the server.
  - **Client Service**: Facilitates user interaction with the web application's functionality.

## Technologies Used

- **Go**: The primary language used for the backend services.
- **HTML/CSS**: Used for the frontend interface.

## Getting Started

To get started with CloudStorage, follow these steps:

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/Pr1t3/CloudStorage.git
   cd CloudStorage
   ```

2. **Start each service**:
   ```bash
   cd Service_name
   go run cmd/main.go
   ```