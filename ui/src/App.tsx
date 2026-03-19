import { createContext, useContext } from "react";
import { RouterProvider } from "@tanstack/react-router";
import { ConfigProvider } from "./contexts/ConfigContext";
import { UserProvider } from "./contexts/UserContext";
import { UIPreferencesProvider } from "./contexts/UIPreferencesContext";
import { SSEProvider } from "./contexts/SSEContext";
import { TimeTrackerProvider } from "./contexts/TimeTrackerContext";
import { ChatProvider } from "./contexts/ChatContext";
import { GlobalTaskProvider } from "./contexts/GlobalTaskContext";
import { OpenCodeProvider } from "./contexts/OpenCodeContext";
import { OpenCodeEventProvider } from "./contexts/OpenCodeEventContext";
import { router } from "./router";

interface ThemeContextType {
	isDark: boolean;
	toggle: (event: React.MouseEvent<HTMLButtonElement>) => void;
}

export const ThemeContext = createContext<ThemeContextType>({
	isDark: false,
	toggle: () => {},
});

declare global {
	interface Document {
		startViewTransition?: (callback: () => void) => {
			ready: Promise<void>;
			finished: Promise<void>;
		};
	}
}

export const useTheme = () => useContext(ThemeContext);

export default function App() {
	return (
		<ConfigProvider>
			<UserProvider>
				<UIPreferencesProvider>
					<OpenCodeProvider>
						<OpenCodeEventProvider>
							<SSEProvider>
								<TimeTrackerProvider>
									<ChatProvider>
										<GlobalTaskProvider>
											<RouterProvider router={router} />
										</GlobalTaskProvider>
									</ChatProvider>
								</TimeTrackerProvider>
							</SSEProvider>
						</OpenCodeEventProvider>
					</OpenCodeProvider>
				</UIPreferencesProvider>
			</UserProvider>
		</ConfigProvider>
	);
}
