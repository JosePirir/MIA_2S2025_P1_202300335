import React, { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

function useQuery() {
  return new URLSearchParams(useLocation().search);
}

function normalizePath(p) {
  if (!p) return '/';
  const parts = p.split('/').filter(Boolean);
  const stack = [];
  for (const part of parts) {
    if (part === '.' || part === '') continue;
    if (part === '..') {
      if (stack.length > 0) stack.pop();
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
  const [dirPerms, setDirPerms] = useState('');

  useEffect(() => {
    const qp = query.get('path') || '/';
    const np = normalizePath(decodeURIComponent(qp));
    setPath(np);
    fetchDir(np);
    fetchStat(np);
    // eslint-disable-next-line
  }, [disk, start, query.get('path')]);

  async function fetchStat(p) {
    if (!disk || !start) return;
    try {
      const res = await onExecute(`statfs -disk=${disk} -start=${start} -path=${p}`);
      const rawLines = (res || '').split('\n').map(l => l.trim()).filter(Boolean);
      const statLine = rawLines.find(l => l.startsWith('STAT|'));
      if (statLine) {
        const parts = statLine.split('|'); // STAT|TYPE|NAME|SIZE|PERMS
        setDirPerms(parts[4] || '');
      } else {
        setDirPerms('');
      }
    } catch {
      setDirPerms('');
    }
  }

  async function fetchDir(p = path) {
    if (!disk || !start) return;
    setLoading(true);
    try {
      const res = await onExecute(`listfs -disk=${disk} -start=${start} -path=${p}`);
      const rawLines = (res || '').split('\n').map(l => l.replace(/\r/g, '').trim()).filter(Boolean);
      // eliminar eco de comando o líneas redundantes
      const useful = rawLines.filter(l => !l.startsWith('>') && l !== disk && l !== p);
      const parsed = useful.map(line => {
        const parts = line.split('|');
        return {
          type: parts[0] || '',
          name: parts[1] || '',
          size: parts[2] || '',
          perms: parts[3] || '' // USAR directamente el campo que envía listfs (permsToString)
        };
      }).filter(it => it.name !== '.' && it.name !== '..');
      setItems(parsed);
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
    navigate(`/browse?disk=${encodeURIComponent(disk)}&start=${encodeURIComponent(start)}&path=${encodeURIComponent(newPath)}`);
    setPath(newPath);
  }

  function goUp() {
    if (!path || path === '/') return;
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

      <h5>
        Ruta: {path} {dirPerms ? <small className="text-muted">· permisos: {dirPerms}</small> : null}
      </h5>

      {loading ? <p>Cargando...</p> : null}

      <table className="table table-dark table-striped">
        <thead>
          <tr><th>Tipo</th><th>Nombre</th><th>Tamaño</th><th>Permisos</th><th>Acción</th></tr>
        </thead>
        <tbody>
          {items.map((it, idx) => (
            <tr key={idx}>
              <td>{it.type}</td>
              <td style={{ wordBreak: 'break-all' }}>{it.name}</td>
              <td>{it.size || '-'}</td>
              <td>{it.perms || '-'}</td>
              <td>
                {it.type === 'DIR' ? (
                  <button className="btn btn-sm btn-outline-light" onClick={() => goInto(it.name)}>Entrar</button>
                ) : (
                  <button className="btn btn-sm btn-outline-light" onClick={() => viewFile(it.name)}>Ver</button>
                )}
              </td>
            </tr>
          ))}
          {(!loading && items.length === 0) && <tr><td colSpan="5">No hay elementos.</td></tr>}
        </tbody>
      </table>

      <div className="mt-3">
        <div className="d-flex justify-content-between align-items-center mb-2">
          <h6 className="mb-0">Contenido de archivo</h6>
          <button className="btn btn-sm btn-outline-light" onClick={() => navigate(`/disco?path=${encodeURIComponent(disk)}`)}>Volver a particiones</button>
        </div>
        <textarea className="form-control bg-dark text-light" value={fileContent} rows={12} readOnly />
      </div>
    </div>
  );
}

export default FileBrowser;