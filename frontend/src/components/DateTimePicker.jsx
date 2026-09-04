import { useEffect, useRef, useState } from 'react';

const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];
const WEEKDAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'];

function pad(n) {
  return String(n).padStart(2, '0');
}

function toLocalValue(date) {
  const d = new Date(date);
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset());
  return d.toISOString().slice(0, 16);
}

function fromLocalValue(value) {
  return value ? new Date(value) : new Date();
}

function formatDisplay(value) {
  if (!value) return 'Select date and time';
  const d = fromLocalValue(value);
  return d.toLocaleString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

function daysInMonth(year, month) {
  return new Date(year, month + 1, 0).getDate();
}

function startWeekday(year, month) {
  return new Date(year, month, 1).getDay();
}

export default function DateTimePicker({ value, onChange, disabled = false }) {
  const containerRef = useRef(null);
  const selected = fromLocalValue(value);
  const [open, setOpen] = useState(false);
  const [viewYear, setViewYear] = useState(selected.getFullYear());
  const [viewMonth, setViewMonth] = useState(selected.getMonth());
  const [hour, setHour] = useState(() => {
    const h = selected.getHours() % 12 || 12;
    return h;
  });
  const [minute, setMinute] = useState(selected.getMinutes());
  const [ampm, setAmpm] = useState(selected.getHours() >= 12 ? 'PM' : 'AM');

  useEffect(() => {
    function handleClick(e) {
      if (containerRef.current && !containerRef.current.contains(e.target)) {
        setOpen(false);
      }
    }
    if (open) document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [open]);

  useEffect(() => {
    if (disabled) setOpen(false);
  }, [disabled]);

  function emitChange(year, month, day, h, m, ap) {
    let hours = h % 12;
    if (ap === 'PM') hours += 12;
    const d = new Date(year, month, day, hours, m, 0, 0);
    onChange(toLocalValue(d));
  }

  function selectDay(day) {
    emitChange(viewYear, viewMonth, day, hour, minute, ampm);
  }

  function updateTime(h, m, ap) {
    emitChange(
      selected.getFullYear(),
      selected.getMonth(),
      selected.getDate(),
      h,
      m,
      ap,
    );
  }

  function prevMonth() {
    if (viewMonth === 0) {
      setViewMonth(11);
      setViewYear((y) => y - 1);
    } else {
      setViewMonth((m) => m - 1);
    }
  }

  function nextMonth() {
    if (viewMonth === 11) {
      setViewMonth(0);
      setViewYear((y) => y + 1);
    } else {
      setViewMonth((m) => m + 1);
    }
  }

  const totalDays = daysInMonth(viewYear, viewMonth);
  const startDay = startWeekday(viewYear, viewMonth);
  const cells = [];
  for (let i = 0; i < startDay; i++) cells.push(null);
  for (let d = 1; d <= totalDays; d++) cells.push(d);

  const selDay = selected.getDate();
  const selMonth = selected.getMonth();
  const selYear = selected.getFullYear();

  return (
    <div className={`datetime-picker${disabled ? ' datetime-picker-disabled' : ''}`} ref={containerRef}>
      <button
        type="button"
        className="datetime-picker-trigger"
        onClick={() => !disabled && setOpen((o) => !o)}
        aria-expanded={open}
        disabled={disabled}
      >
        <span className="datetime-picker-icon" aria-hidden="true" />
        <span>{formatDisplay(value)}</span>
      </button>

      {open && (
        <div className="datetime-picker-popover">
          <div className="datetime-picker-calendar">
            <div className="datetime-picker-header">
              <button type="button" className="datetime-nav-btn" onClick={prevMonth} aria-label="Previous month">‹</button>
              <span className="datetime-picker-month">{MONTHS[viewMonth]} {viewYear}</span>
              <button type="button" className="datetime-nav-btn" onClick={nextMonth} aria-label="Next month">›</button>
            </div>
            <div className="datetime-picker-weekdays">
              {WEEKDAYS.map((d) => <span key={d}>{d}</span>)}
            </div>
            <div className="datetime-picker-days">
              {cells.map((day, i) => (
                <button
                  key={i}
                  type="button"
                  className={[
                    'datetime-day',
                    day === null ? 'empty' : '',
                    day === selDay && viewMonth === selMonth && viewYear === selYear ? 'selected' : '',
                  ].filter(Boolean).join(' ')}
                  disabled={day === null}
                  onClick={() => day && selectDay(day)}
                >
                  {day ?? ''}
                </button>
              ))}
            </div>
          </div>

          <div className="datetime-picker-time">
            <span className="datetime-time-label">Time</span>
            <div className="datetime-time-inputs">
              <select
                value={hour}
                onChange={(e) => {
                  const h = Number(e.target.value);
                  setHour(h);
                  updateTime(h, minute, ampm);
                }}
              >
                {Array.from({ length: 12 }, (_, i) => i + 1).map((h) => (
                  <option key={h} value={h}>{h}</option>
                ))}
              </select>
              <span className="datetime-time-sep">:</span>
              <select
                value={minute}
                onChange={(e) => {
                  const m = Number(e.target.value);
                  setMinute(m);
                  updateTime(hour, m, ampm);
                }}
              >
                {Array.from({ length: 60 }, (_, i) => i).map((m) => (
                  <option key={m} value={m}>{pad(m)}</option>
                ))}
              </select>
              <select
                value={ampm}
                onChange={(e) => {
                  const ap = e.target.value;
                  setAmpm(ap);
                  updateTime(hour, minute, ap);
                }}
              >
                <option value="AM">AM</option>
                <option value="PM">PM</option>
              </select>
            </div>
            <button type="button" className="btn btn-primary datetime-done-btn" onClick={() => setOpen(false)}>
              Done
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
