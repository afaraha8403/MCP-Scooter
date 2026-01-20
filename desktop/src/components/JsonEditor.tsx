import { useEffect, useRef } from 'react';
import { createJSONEditor } from 'vanilla-jsoneditor';
import type { JSONEditor, JSONEditorPropsOptional, Content } from 'vanilla-jsoneditor';

interface JsonEditorProps extends Omit<JSONEditorPropsOptional, 'content'> {
  content: Content;
  onChange?: (content: Content, previousContent: Content, status: { contentErrors: any; patchResult: any }) => void;
  onBlur?: () => void;
  className?: string;
  height?: string | number;
  dark?: boolean;
}

export const JsonEditor = ({ 
  content, 
  onChange, 
  onBlur,
  className, 
  height = '300px',
  dark = false,
  ...restProps 
}: JsonEditorProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<JSONEditor | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    editorRef.current = createJSONEditor({
      target: containerRef.current,
      props: {
        content,
        onChange,
        ...restProps
      }
    });

    return () => {
      if (editorRef.current) {
        editorRef.current.destroy();
        editorRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    if (editorRef.current) {
      editorRef.current.updateProps({
        content,
        onChange,
        ...restProps
      });
    }
  }, [content, onChange, restProps]);

  return (
    <div 
      ref={containerRef} 
      className={`json-editor-container ${dark ? 'jse-theme-dark' : ''} ${className || ''}`}
      style={{ height }}
      onBlur={onBlur}
    />
  );
};
