// --- Reusable Components ---
const StatCard = ({ title, value, icon, change }) => (
  <div className="bg-white p-6 rounded-lg shadow-md">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm font-medium text-gray-500">{title}</p>
        <p className="text-2xl font-bold text-gray-800">{value}</p>
      </div>
      <div className="bg-indigo-100 text-indigo-600 p-3 rounded-full">
        {icon}
      </div>
    </div>
    {change && <p className="text-sm text-gray-500 mt-2">{change}</p>}
  </div>
);

export { StatCard };