import React from 'react';

const OutputArea = ({ output = '', onClear }) => {
  // Separar bloques por doble salto de línea y eliminar vacíos
  const blocks = output.split('\n\n').filter(Boolean);

  return (
    <div className="card shadow-lg border-0">
      {/* Header */}
      <div className="card-header d-flex justify-content-between align-items-center bg-secondary text-white">
        <h5 className="mb-0 d-flex align-items-center">
          <i className="bi bi-terminal-fill me-2"></i>
          Área de Salida de Comandos
        </h5>
        <button
          type="button"
          className="btn btn-sm btn-outline-light d-flex align-items-center"
          onClick={onClear}
        >
          <i className="bi bi-trash me-1"></i>
          Limpiar
        </button>
      </div>

      {/* Body */}
      <div
        className="card-body p-2 bg-dark rounded-bottom overflow-auto"
        style={{
          height: '500px',
          fontFamily: 'monospace',
          fontSize: '0.9rem',
          color: 'white'
        }}
      >
        {blocks.length > 0 ? (
          blocks.map((block, index) => (
            <div
              key={index}
              className="mb-2 p-2 rounded shadow-sm"
              style={{
                backgroundColor: 'rgba(255,255,255,0.07)',
                border: '1px solid rgba(255,255,255,0.25)',
                whiteSpace: 'pre-wrap',
                color: 'white'
              }}
            >
              {block
                .split('\n')
                .filter(Boolean) // eliminar líneas vacías
                .map((line, i) => (
                  <div key={i}>{line}</div>
                ))}
            </div>
          ))
        ) : (
          <div className="text-muted px-2">
            Ejecute algunos comandos para ver la salida aquí...
          </div>
        )}
      </div>
    </div>
  );
};

export default OutputArea;