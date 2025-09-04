import React, { useState } from 'react';
import Header from './components/Header';
import CommandArea from './components/CommandArea';
import OutputArea from './components/OutputArea';
import { executeCommands } from './services/api';

function App() {
  const [output, setOutput] = useState('');
  const [isExecuting, setIsExecuting] = useState(false);

  const handleExecuteCommands = async (commands) => {
    setIsExecuting(true);
    try {
      const result = await executeCommands(commands);
      setOutput(prev => prev + '\n' + result);
    } catch (error) {
      setOutput(prev => prev + '\nError: ' + error.message);
    } finally {
      setIsExecuting(false);
    }
  };

  const handleClearOutput = () => {
    setOutput('');
  };

  return (
    <div className="App bg-dark text-light min-vh-100">
      {/* Header */}
      <Header />

      {/* Contenido */}
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
    </div>
  );
}

export default App;