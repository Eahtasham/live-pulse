import { ReactNode } from "react";

interface PageWrapperProps {
  children: ReactNode;
  className?: string;
}

export function PageWrapper({ children, className = "" }: PageWrapperProps) {
  return (
    <div className={`mx-auto w-full max-w-7xl px-6 py-6 lg:px-8 ${className}`}>
      {children}
    </div>
  );
}
