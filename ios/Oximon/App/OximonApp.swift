import SwiftUI

@main
struct OximonApp: App {
    var body: some Scene {
        WindowGroup {
            #if targetEnvironment(simulator)
            DashboardView(manager: SimulatedOximeterManager())
                .preferredColorScheme(.dark)
            #else
            DashboardView(manager: OximeterManager())
                .preferredColorScheme(.dark)
            #endif
        }
    }
}
