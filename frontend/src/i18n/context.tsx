import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import { en, type TranslationKeys } from './en';
import { zh } from './zh';

type Lang = 'en' | 'zh';

const translations: Record<Lang, TranslationKeys> = { en, zh };

const STORAGE_KEY = 'lc-lang';

function detectLang(): Lang {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'en' || stored === 'zh') return stored;
  return navigator.language.startsWith('zh') ? 'zh' : 'en';
}

function resolve(obj: unknown, path: string): string {
  const keys = path.split('.');
  let cur: unknown = obj;
  for (const k of keys) {
    if (cur == null || typeof cur !== 'object') return path;
    cur = (cur as Record<string, unknown>)[k];
  }
  return typeof cur === 'string' ? cur : path;
}

interface I18nContextValue {
  lang: Lang;
  setLang: (lang: Lang) => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(detectLang);

  const setLang = useCallback((l: Lang) => {
    localStorage.setItem(STORAGE_KEY, l);
    setLangState(l);
  }, []);

  useEffect(() => {
    // Listen for portal language changes
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (detail?.lang === 'en' || detail?.lang === 'zh') {
        setLangState(detail.lang);
      }
    };
    window.addEventListener('lc:langchange', handler);
    return () => window.removeEventListener('lc:langchange', handler);
  }, []);

  const t = useCallback((key: string, vars?: Record<string, string | number>): string => {
    let result = resolve(translations[lang], key);
    if (vars) {
      for (const [k, v] of Object.entries(vars)) {
        result = result.replace(`{${k}}`, String(v));
      }
    }
    return result;
  }, [lang]);

  return (
    <I18nContext.Provider value={{ lang, setLang, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error('useI18n must be used within I18nProvider');
  return ctx;
}
