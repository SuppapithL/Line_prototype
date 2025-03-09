"use client";

import { useState } from "react";

export default function ChatInput() {
  const [chatInputValue, setChatInputValue] = useState("");

  const sendMessage = (message: string) => {
    if (message.trim() === "") return; // Ignore empty messages
    console.log("Sending message:", message);
    // Here, you'd send it to the chatbot backend or process it
  };

  return (
    <div
      style={{
        position: "fixed",
        bottom: "60px",
        left: 0,
        right: 0,
        padding: "10px",
        backgroundColor: "#fff",
        borderTop: "1px solid #ddd",
        zIndex: 1000,
      }}
    >
      <div style={{ position: "relative" }}>
        <input
          type="text"
          placeholder="Type a message..."
          value={chatInputValue}
          onChange={(e) => setChatInputValue(e.target.value)}
          onKeyUp={(e) => {
            if (e.key === "Enter") {
              sendMessage(chatInputValue);
              setChatInputValue(""); // Clear input after sending
            }
          }}
          style={{
            width: "100%",
            padding: "10px",
            borderRadius: "4px",
            border: "1px solid #ccc",
          }}
        />
      </div>
    </div>
  );
}
