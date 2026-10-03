import { createContext, useContext } from 'react';

export const ConfirmContext = createContext(null);

// const ok = await confirm({ title, message, action: 'Delete' });
export const useConfirm = () => useContext(ConfirmContext);
