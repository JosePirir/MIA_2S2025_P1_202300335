import React, { useState } from "react";

const CommandArea = ({ onExecute, isExecuting }) => {
  const [commands, setCommands] = useState("");

  const handleFileUpload = (event) => {
    const file = event.target.files[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        setCommands(e.target.result);
      };
      reader.readAsText(file);
    }
  };

  const handleExecute = () => {
    if (commands.trim()) {
      onExecute(commands);
    }
  };

  return (
    <div className="card shadow-lg border-0">
      {/* Header */}
      <div className="card-header bg-success text-white">
        <h5 className="mb-0 d-flex align-items-center">
          <i className="bi bi-terminal me-2"></i>
          Área de Comandos
        </h5>
      </div>

      {/* Body */}
      <div className="card-body d-flex flex-column gap-4">
        {/* Sección de archivo */}
        <div>
          <label htmlFor="fileInput" className="form-label fw-bold">
            <i className="bi bi-upload me-2 text-success"></i>
            Subir archivo de comandos
          </label>
          <input
            type="file"
            id="fileInput"
            accept=".txt,.sh"
            onChange={handleFileUpload}
            className="form-control mb-2"
          />
        </div>

        {/* Sección de textarea */}
        <div>
          <label htmlFor="commandsTextarea" className="form-label fw-bold">
            <i className="bi bi-pencil-square me-2 text-success"></i>
            Escribir o editar comandos
          </label>
          <textarea
            id="commandsTextarea"
            rows="14"
            value={commands}
            onChange={(e) => setCommands(e.target.value)}
            placeholder="Ingrese los comandos aquí..."
            className="form-control font-monospace border-2"
          />
        </div>
        <button
            type="button"
            className={`btn btn-lg w-100 text-white fw-bold ${
              isExecuting || !commands.trim()
                ? "btn-secondary disabled"
                : "btn-success"
            }`}
            onClick={handleExecute}
            disabled={isExecuting || !commands.trim()}
          >
            {isExecuting ? (
              <>
                <span
                  className="spinner-border spinner-border-sm me-2"
                  role="status"
                  aria-hidden="true"
                ></span>
                Ejecutando...
              </>
            ) : (
              <>
                <i className="bi bi-play-fill me-2"></i>
                Ejecutar
              </>
            )}
          </button>
      </div>
    </div>
  );
};

export default CommandArea;
