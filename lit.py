import streamlit as st
import psycopg2
import pandas as pd
import base64
from io import BytesIO
from PIL import Image
import requests
import certifi

# ---------------------------
# Database Connection Settings
# ---------------------------
DB_CONFIG = {
    "dbname": "file_manager",
    "user": "postgres",
    "password": "Suppapith2",  # replace with your actual password
    "host": "localhost",
    "port": "5432"
}

def get_db_connection():
    """Establish a connection to the PostgreSQL database."""
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        return conn
    except Exception as e:
        st.error(f"Error connecting to database: {e}")
        return None

# ---------------------------
# Database Functions
# ---------------------------
def create_user(user_id, username, password):
    """Register a new user."""
    conn = get_db_connection()
    if conn:
        cur = conn.cursor()
        cur.execute(
            "INSERT INTO users (user_id, username, password) VALUES (%s, %s, %s)",
            (user_id, username, password)
        )
        conn.commit()
        conn.close()

def validate_user(user_id_or_username, password):
    """Validate if the user exists and password matches."""
    conn = get_db_connection()
    if conn:
        cur = conn.cursor()
        cur.execute("""
            SELECT user_id, username FROM users 
            WHERE (user_id = %s OR username = %s) AND password = %s
        """, (user_id_or_username, user_id_or_username, password))
        user = cur.fetchone()
        conn.close()
        return user  # Returns (user_id, username) if valid, or None if invalid
    return None

def get_user_themes(user_id):
    """Fetch all themes for a user."""
    conn = get_db_connection()
    if conn:
        cur = conn.cursor()
        cur.execute("SELECT DISTINCT theme FROM line_01 WHERE user_id = %s", (user_id,))
        themes = [row[0] for row in cur.fetchall()]
        conn.close()
        return themes
    return []

def display_pdf(file_content):
    pdf_base64 = base64.b64encode(file_content).decode("utf-8")
    pdf_display = f'<iframe src="data:application/pdf;base64,{pdf_base64}" width="700" height="500" type="application/pdf"></iframe>'
    st.markdown(pdf_display, unsafe_allow_html=True)
    
def display_image(file_content):
    try:
        image = Image.open(BytesIO(file_content))
        st.image(image, caption="Preview", use_container_width=True)
    except Exception as e:
        st.error(f"❌ Failed to load image: {e}")

def get_files_in_theme(user_id, theme):
    """Get files for a specific theme."""
    conn = get_db_connection()
    if conn:
        cur = conn.cursor()
        cur.execute(
            "SELECT file_name, created_at, file_content FROM line_01 WHERE user_id = %s AND theme = %s",
            (user_id, theme)
        )
        files = cur.fetchall()
        conn.close()
        return pd.DataFrame(files, columns=["File Name", "Created At", "File Content"])
    return pd.DataFrame()

def get_file_data(file_url):
    """Fetch file content from R2 storage with certificate verification."""
    try:
        response = requests.get(file_url, verify=certifi.where())
        response.raise_for_status()  # Raise an exception for HTTP errors
        return response.content
    except requests.exceptions.RequestException as e:
        st.error(f"Error fetching file: {e}")
        st.write(f"File URL: {file_url}")  # Add debug output
        return None

# ---------------------------
# Session State Initialization
# ---------------------------
if "page" not in st.session_state:
    st.session_state.page = "login"

if "user_id" not in st.session_state:
    st.session_state.user_id = None

if "username" not in st.session_state:
    st.session_state.username = None  # เพิ่มอันนี้กัน error

# Assign the current page from session state
page = st.session_state.page

# ---------------------------
# Login Page
# ---------------------------
if page == "login":
    st.title("Login to Access File Manager")
    
    user_id_or_username = st.text_input("Enter your User ID or Username")
    password = st.text_input("Enter your Password", type="password")
    
    if st.button("Login"):
        if user_id_or_username and password:
            user = validate_user(user_id_or_username, password)
            
            if user:
                user_id, username = user  
                st.session_state.user_id = user_id
                st.session_state.username = username
                st.session_state.page = "dashboard"  # Change page
                st.success(f"Login successful! Welcome, {username}!")
                st.rerun()  # Force rerun to show dashboard
            else:
                st.error("Invalid User ID/Username or Password")
        else:
            st.warning("Please enter both User ID/Username and Password")

    if st.button("Register New Account"):
        st.session_state.page = "register"
        st.rerun()  # Navigate to register page

# ---------------------------
# Dashboard Page
# ---------------------------
elif page == "dashboard":
    if st.session_state.user_id is None:
        st.warning("You need to log in first!")
        st.session_state.page = "login"
        st.rerun()  # Redirect to login page
    else:
        user_id = st.session_state.user_id
        username = st.session_state.username

        st.title(f"📂 {username}'s File Manager 🚀")

        themes = get_user_themes(user_id)

        if themes:
            selected_theme = st.selectbox("Select a folder (theme)", themes)

            if selected_theme:
                df_files = get_files_in_theme(user_id, selected_theme)
                if not df_files.empty:
                    selected_file = st.selectbox("Select a file", df_files["File Name"].tolist())
                    if selected_file:
                        # Get the file content (URL or path)
                        file_url = df_files[df_files["File Name"] == selected_file]["File Content"].values[0]

                        # Check the file extension and display appropriately
                        if file_url.endswith(".txt"):
                            # If the file is a text file, fetch the content and display
                            file_data = get_file_data(file_url)
                            if file_data:
                                text_data = file_data.decode("utf-8")
                                
                                # Display the text content without allowing editing
                                st.text(f"📄 File Content:\n{text_data}")
                                
                                # Add a download button for the text file
                                st.download_button(
                                    label="Download TXT",
                                    data=file_data,
                                    file_name=f"{selected_file}.txt",  # Add the .txt extension
                                    mime="text/plain"
                                )            
                        elif file_url.endswith((".jpg", ".jpeg", ".png")):
                            # If the file is an image, display the image
                            file_data = get_file_data(file_url)
                            if file_data:
                                display_image(file_data)

                                # Add a download button for the image file
                                st.download_button(
                                    label="Download Image",
                                    data=file_data,
                                    file_name=f"{selected_file}.jpg" if selected_file.endswith((".jpg", ".jpeg")) else f"{selected_file}.png",  # Add the correct extension
                                    mime="image/jpeg" if selected_file.endswith((".jpg", ".jpeg")) else "image/png"
                                )
                        
                        elif file_url.endswith(".pdf"):
                            # If the file is a PDF, display the PDF
                            file_data = get_file_data(file_url)
                            if file_data:
                                display_pdf(file_data)

                                # Add a download button for the PDF file
                                st.download_button(
                                    label="Download PDF",
                                    data=file_data,
                                    file_name=f"{selected_file}.pdf",  # Add the .pdf extension
                                    mime="application/pdf"
                                )

                        else:
                            # Handle unsupported file types
                            st.warning("🔹 Cannot preview this file type. Try downloading it.")
                else:
                    st.warning("No files found in this folder")
        else:
            st.warning("No folders found for your account")

        # Logout Button
        if st.button("Logout"):
            st.session_state.user_id = None
            st.session_state.page = "login"
            st.rerun()  # Refresh to login page

# ---------------------------
# Register Page
# ---------------------------
elif page == "register":
    st.title("Register for File Manager")
    
    new_user_id = st.text_input("Enter a new User ID")
    new_username = st.text_input("Enter a new Username")
    new_password = st.text_input("Enter a new Password", type="password")
    confirm_password = st.text_input("Confirm your Password", type="password")
    
    if st.button("Register"):
        if new_user_id and new_username and new_password and confirm_password:
            if new_password == confirm_password:
                try:
                    create_user(new_user_id, new_username, new_password)
                    st.success("Registration successful! Please log in.")
                    st.session_state.page = "login"
                    st.rerun()  # Refresh to login page
                except Exception as e:
                    st.error(f"Error during registration: {e}")
            else:
                st.error("Passwords do not match")
        else:
            st.warning("Please fill in all fields")

    if st.button("Back to Login"):
        st.session_state.page = "login"
        st.rerun()  # Refresh to login page
