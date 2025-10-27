import React, { useState } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Header from './components/Header';
import CommandArea from './components/CommandArea';
import OutputArea from './components/OutputArea';
import LoginPage from './components/LoginPage';
import DisksPage from './components/DisksPage';
import PartitionsPage from './components/PartitionsPage';
import FileBrowser from './components/FileBrowser';
import { executeCommands } from './services/api';

function App() {
  const [output, setOutput] = useState('');
  const [isExecuting, setIsExecuting] = useState(false);

  const handleExecuteCommands = async (commands) => {
    setIsExecuting(true);
    try {
      const result = await executeCommands(commands); // executeCommands debe devolver string
      setOutput(prev => prev + '\n' + result);
      return result; // <- importante
    } catch (error) {
      setOutput(prev => prev + '\nError: ' + error.message);
      return 'ERROR: ' + (error.message || '');
    } finally {
      setIsExecuting(false);
    }
  };

  const handleClearOutput = () => {
    setOutput('');
  };

  const Main = () => (
    <div className="container-fluid mt-4">
      <div className="row">
        <div className="col-md-6 mb-4">
          <CommandArea 
            onExecute={handleExecuteCommands}
            isExecuting={isExecuting}
          />
        </div>
        <div className="col-md-6 mb-4">
          <OutputArea 
            output={output}
            onClear={handleClearOutput}
          />
        </div>
      </div>
    </div>
  );

  return (
    <BrowserRouter>
      <div className="App bg-dark text-light min-vh-100">
        <Header onExecute={handleExecuteCommands} />  {/* <--- aquí */}
        <Routes>
          <Route path="/" element={<Main />} />
          <Route path="/login" element={<LoginPage onExecute={handleExecuteCommands} />} />
          <Route path="/discos" element={<DisksPage onExecute={handleExecuteCommands} />} />
          <Route path="/disco" element={<PartitionsPage onExecute={handleExecuteCommands} />} />
          <Route path="/browse" element={<FileBrowser onExecute={handleExecuteCommands} />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}

export default App;