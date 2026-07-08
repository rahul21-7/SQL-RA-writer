import os
import zipfile
import re
import requests
from tqdm import tqdm

GD_FILE_ID = "1403EGqzIDoHMdQF4c9Bkyl7dZLZ5Wt6J"
ZIP_PATH = "spider.zip"
EXTRACT_DIR = "data"

def download_file_from_google_drive(file_id, destination):
    print(f"Downloading Spider dataset zip (ID: {file_id}) from Google Drive...")
    url = "https://docs.google.com/uc?export=download"
    
    session = requests.Session()
    response = session.get(url, params={'id': file_id}, stream=True)
    
    # Check if we hit the virus scan warning page
    if "Virus scan warning" in response.text or "too large for Google to scan" in response.text:
        print("Redirect/warning detected. Bypassing Google Drive virus scan check...")
        
        # Find the form action URL
        action_match = re.search(r'action="([^"]+)"', response.text)
        action_url = action_match.group(1) if action_match else "https://drive.usercontent.google.com/download"
        
        # Parse all hidden input fields (name and value attributes)
        inputs = re.findall(r'name="([^"]+)"\s+value="([^"]*)"', response.text)
        params = {name: val for name, val in inputs}
        
        # Ensure we have the necessary parameters
        params['id'] = file_id
        params['export'] = 'download'
        params['confirm'] = 't' # 't' forces the bypass
        
        # Perform follow-up request to get the actual zip file
        response = session.get(action_url, params=params, stream=True)
        
    total_size = int(response.headers.get('content-length', 0))
    block_size = 32768
    
    t = tqdm(total=total_size if total_size else None, unit='B', unit_scale=True, desc="Downloading")
    with open(destination, 'wb') as f:
        for chunk in response.iter_content(block_size):
            if chunk:
                f.write(chunk)
                t.update(len(chunk))
    t.close()
    return True

def extract_database_folder(zip_path, extract_dir):
    print("Extracting 'database' folder to 'data/database'...")
    with zipfile.ZipFile(zip_path, 'r') as zip_ref:
        members = zip_ref.namelist()
        db_members = [m for m in members if m.startswith('spider_data/database/')]
        for member in tqdm(db_members, desc="Extracting"):
            relative_path = member.replace("spider_data/", "", 1)
            target_path = os.path.join(extract_dir, relative_path)
            
            if member.endswith('/'):
                os.makedirs(target_path, exist_ok=True)
            else:
                os.makedirs(os.path.dirname(target_path), exist_ok=True)
                with zip_ref.open(member) as source, open(target_path, "wb") as target:
                    target.write(source.read())
    print("Extraction completed successfully!")

def main():
    if os.path.exists(ZIP_PATH):
        if not zipfile.is_zipfile(ZIP_PATH):
            print(f"Removing invalid/truncated local file '{ZIP_PATH}'...")
            try:
                os.remove(ZIP_PATH)
            except Exception as e:
                print(f"[WARNING] Could not remove '{ZIP_PATH}': {e}")
        else:
            print("Valid spider.zip already exists locally, skipping download.")

    if not os.path.exists(ZIP_PATH):
        success = download_file_from_google_drive(GD_FILE_ID, ZIP_PATH)
        if not success:
            print("[ERROR] Download failed.")
            return
        
    extract_database_folder(ZIP_PATH, EXTRACT_DIR)
    
    if os.path.exists(ZIP_PATH):
        print("Cleaning up spider.zip...")
        os.remove(ZIP_PATH)
        print("Cleanup completed.")

if __name__ == "__main__":
    main()
