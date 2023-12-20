
import React, { useState, useEffect } from 'react';
import './App.css'; // Import file CSS

function App() {
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [webSocket, setWebSocket] = useState(null);

  useEffect(() => {
    const ws = new WebSocket('ws://localhost:8080/ws');
    setWebSocket(ws);

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      setMessages((prevMessages) => [...prevMessages, message]);
    };

    return () => {
      ws.close();
    };
  }, []);

  const handleInputChange = (e) => {
    setInputMessage(e.target.value);
  };

  const handleSendMessage = () => {
    if (inputMessage.trim() === '') {
      return;
    }

    const message = {
      username: 'User',
      text: inputMessage,
    };

    // Send the message to the WebSocket server
    if (webSocket && webSocket.readyState === WebSocket.OPEN) {
      webSocket.send(JSON.stringify(message));
    }

    // Clear the input field
    setInputMessage('');
  };

  return (
    <div className="App">
      <h1>Chat App</h1>
      <div className="ChatContainer">
        {messages.map((message, index) => (
          <div key={index} className="Message">
            <span className="Username">{message.username}:</span> {message.text}
          </div>
        ))}
      </div>
      <div className="InputContainer">
        <input
          type="text"
          value={inputMessage}
          onChange={handleInputChange}
          placeholder="Type your message..."
        />
        <button onClick={handleSendMessage}>Send</button>
      </div>
    </div>
  );
}

export default App;
