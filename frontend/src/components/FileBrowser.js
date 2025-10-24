import React, { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

function useQuery() {
  return new URLSearchParams(useLocation().search);
}

function normalizePath(p) {
  if (!p) return '/';
  // Considerar rutas absolutas y relativas, resolver '.' y '..'
  const isAbsolute = p.startsWith('/');
  const parts = p.split('/').filter(Boolean);
  const stack = [];
  for (const part of parts) {
    if (part === '.' || part === '') continue;
    if (part === '..') {
      if (stack.length > 0) stack.pop();
      // si stack vacío y es ruta relativa, podemos ignorar .. (no subir más)
      continue;
    }
    stack.push(part);
  }
  return '/' + stack.join('/');
}

function FileBrowser({ onExecute }) {
  const query = useQuery();
  const navigate = useNavigate();
  const disk = query.get('disk') || '';
  const start = query.get('start') || '';
  const rawPath = query.get('path') || '/';
  const [path, setPath] = useState(normalizePath(decodeURIComponent(rawPath)));
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [fileContent, setFileContent] = useState('');

  useEffect(() => {
    const qp = query.get('path') || '/';
    const np = normalizePath(decodeURIComponent(qp));
    setPath(np);
    load(np);
    // eslint-disable-next-line
  }, [disk, start, query.get('path')]);

  async function load(p = path) {
    if (!disk || !start) return;
    setLoading(true);
    try {
      const res = await onExecute(`listfs -disk=${disk} -start=${start} -path=${p}`);
      const lines = (res || '').split('\n').map(l => l.trim()).filter(Boolean);
      const parsed = lines.map(line => {
        const parts = line.split('|');
        return {
          type: parts[0] || '',
          name: parts[1] || '',
          size: parts[2] || ''
        };
      })
      // Filtrar entradas especiales "." y ".."
      const filtered = parsed.filter(it => it.name !== '.' && it.name !== '..');
      setItems(filtered);
      setFileContent('');
    } catch (err) {
      setItems([]);
      setFileContent(`Error: ${err.message || String(err)}`);
    } finally {
      setLoading(false);
    }
  }

  function goInto(name) {
    const candidate = path === '/' ? `/${name}` : `${path}/${name}`;
    const newPath = normalizePath(candidate);
    // actualizar URL de forma consistente
    navigate(`/browse?disk=${encodeURIComponent(disk)}&start=${encodeURIComponent(start)}&path=${encodeURIComponent(newPath)}`);
    setPath(newPath);
    // load será llamado por useEffect
  }

  function goUp() {
    if (!path || path === '/' ) return;
    const parts = path.split('/').filter(Boolean);
    parts.pop();
    const newPath = '/' + parts.join('/');
    const normalized = normalizePath(newPath);
    navigate(`/browse?disk=${encodeURIComponent(disk)}&start=${encodeURIComponent(start)}&path=${encodeURIComponent(normalized)}`);
    setPath(normalized);
  }

  async function viewFile(name) {
    const filePath = path === '/' ? `/${name}` : `${path}/${name}`;
    const normalized = normalizePath(filePath);
    try {
      const res = await onExecute(`showfile -disk=${disk} -start=${start} -path=${normalized}`);
      setFileContent(res || '');
    } catch (err) {
      setFileContent(`Error: ${err.message || String(err)}`);
    }
  }

  return (
    <div className="container mt-4">
      <div className="d-flex justify-content-between mb-2">
        <div>
          <button className="btn btn-secondary me-2" onClick={() => navigate(-1)}>Volver</button>
          <button className="btn btn-outline-light me-2" onClick={goUp}>Subir</button>
        </div>
        <div><strong>Disco:</strong> {disk}</div>
      </div>

      <h5>Ruta: {path}</h5>

      {loading ? <p>Cargando...</p> : null}

      <table className="table table-dark table-striped">
        <thead>
          <tr><th>Tipo</th><th>Nombre</th><th>Tamaño</th><th>Acción</th></tr>
        </thead>
        <tbody>
          {items.map((it, idx) => (
            <tr key={idx}>
              <td>{it.type}</td>
              <td style={{wordBreak: 'break-all'}}>{it.name}</td>
              <td>{it.size || '-'}</td>
              <td>
                {it.type === 'DIR' ? (
                  <button className="btn btn-sm btn-outline-light" onClick={() => goInto(it.name)}>Entrar</button>
                ) : (
                  <button className="btn btn-sm btn-outline-light" onClick={() => viewFile(it.name)}>Ver</button>
                )}
              </td>
            </tr>
          ))}
          {(!loading && items.length === 0) && <tr><td colSpan="4">No hay elementos.</td></tr>}
        </tbody>
      </table>

      <div className="mt-3">
        <h6>Contenido de archivo</h6>
        <textarea className="form-control bg-dark text-light" value={fileContent} rows={12} readOnly />
      </div>
    </div>
  );
}

export default FileBrowser;