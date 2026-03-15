# React Native Mobile

## Stack

| Component | Default |
|-----------|---------|
| Framework | React Native |
| Development | Expo (managed workflow) |
| Navigation | React Navigation |
| State | Zustand |

## Decision: Expo vs Bare

- **Expo**: Standard features, rapid prototyping, OTA updates, managed builds
- **Bare**: Custom native modules, advanced native features

## Conventions

- Type-safe navigation with `RootStackParamList` type and `NativeStackScreenProps`
- Platform-specific code: `Platform.select()` or `.ios.tsx`/`.android.tsx` files
- `FlatList` for long lists (never `ScrollView`), with `removeClippedSubviews` and `getItemLayout`
- `expo-image` for optimized image loading with placeholder and transitions
- Type-safe `AsyncStorage` wrapper with generic `get<T>`/`set<T>`

## Critical Rules

- Type navigation params
- Request permissions before accessing native APIs
- Test on both iOS and Android
- Use `FlatList` for long lists
- Handle safe areas
