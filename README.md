The dataset can be downloaded from `[Yale Spider 1.0](https://yale-lily.github.io/spider)`

add `config.json` file in the root of your directory in this format

```
{
  "driver": "postgres",
  "dsn": "host=localhost port=5432 user=postgres password=your-password dbname=myproject sslmode=disable",
  "dialect": "postgres"
}
```

To use this project with PostgreSQL, follow these steps to configure your environment and database.

1. *Database Setup*
    1. *Install PostgreSQL*: Ensure you have PostgreSQL installed and running on your system.
    2. *Create Database*: Create a database named `myproject`.(or change the config.json file accordingly)
    ```
    CREATE DATABASE myproject;
    ```
    3. *Initialize Tables*: Add the required tables to your database. You can find the necessary SQL schema or relational algebra expressions by following the environment setup below.
2. *Environment Setup*
    Create and Activate a Virtual Environment
    ```
    # Create the environment
    python -m venv venv

    # Activate it (Windows)
    .\venv\Scripts\activate

    # Activate it (macOS/Linux)
    source venv/bin/activate
    ```

    *To install the correct GPU-enabled versions of Torch and Unsloth, run:*
    `pip install torch==2.5.1 torchvision==0.20.1 torchaudio==2.5.1 --index-url https://download.pytorch.org/whl/cu121`

    *Install remaining dependenciies:*
    `pip install -r requirements.txt`

    Or just run the `setup.bat` file by double clicking it in file explorer and it will create a virtual environment and install all the dependencies
